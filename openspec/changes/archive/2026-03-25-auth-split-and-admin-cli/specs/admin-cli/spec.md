## ADDED Requirements

### Requirement: Cobra root command
The backend binary SHALL use `github.com/spf13/cobra` as its command framework. The root command (`vynilino`) SHALL print help when called with no subcommand and SHALL NOT start the HTTP server on its own.

#### Scenario: No subcommand prints help and exits 0
- **WHEN** the binary is invoked with no arguments
- **THEN** the binary SHALL print the help message listing available subcommands and exit with code 0

### Requirement: `serve` subcommand starts the HTTP server
The system SHALL expose a `serve` subcommand that starts the HTTP server with identical behavior to the previous `main` entrypoint. All existing environment variables and the `--check-migrations` flag SHALL be supported.

#### Scenario: Server starts on valid config
- **WHEN** `vynilino serve` is invoked with a valid configuration (env vars set, DB accessible)
- **THEN** the HTTP server SHALL start, log "server listening", and begin accepting requests

#### Scenario: Migration check flag exits cleanly
- **WHEN** `vynilino serve --check-migrations` is invoked and migrations are up to date
- **THEN** the binary SHALL log "migrations up to date" and exit with code 0

#### Scenario: Migration check flag exits with error on pending migrations
- **WHEN** `vynilino serve --check-migrations` is invoked and pending migrations exist
- **THEN** the binary SHALL log an error describing the pending migrations and exit with code 1

### Requirement: `user list` subcommand
The system SHALL expose a `vynilino user list` command that prints all users to stdout in a human-readable tabular format (ID, email, status, created-at).

#### Scenario: Users listed to stdout
- **WHEN** `vynilino user list` is invoked with a valid DB path
- **THEN** the command SHALL print a table of all users and exit with code 0

#### Scenario: Empty user list
- **WHEN** `vynilino user list` is invoked and no users exist
- **THEN** the command SHALL print a header row and exit with code 0

### Requirement: `user add` subcommand
The system SHALL expose a `vynilino user add --email <email>` command that creates a new user account. The password SHALL be prompted interactively (masked) or read from stdin when `--password-stdin` is passed.

#### Scenario: User created with interactive password
- **WHEN** `vynilino user add --email user@example.com` is invoked and a valid password is entered at the prompt
- **THEN** the command SHALL create the user, print the new user ID, and exit with code 0

#### Scenario: User created with stdin password
- **WHEN** `vynilino user add --email user@example.com --password-stdin` is invoked with a password on stdin
- **THEN** the command SHALL create the user, print the new user ID, and exit with code 0

#### Scenario: Duplicate email rejected
- **WHEN** `vynilino user add` is invoked with an email that already exists
- **THEN** the command SHALL print an error "email already in use" and exit with code 1

#### Scenario: Weak password rejected
- **WHEN** `vynilino user add` is invoked with a password shorter than 12 characters
- **THEN** the command SHALL print an error "password too weak" and exit with code 1

### Requirement: `user deactivate` subcommand
The system SHALL expose a `vynilino user deactivate --email <email>` command that sets the target user's status to inactive, preventing login.

#### Scenario: User deactivated successfully
- **WHEN** `vynilino user deactivate --email user@example.com` is invoked for an existing active user
- **THEN** the command SHALL mark the user inactive and print "user deactivated" and exit with code 0

#### Scenario: Already inactive user
- **WHEN** `vynilino user deactivate` is invoked for a user already inactive
- **THEN** the command SHALL print "user already inactive" and exit with code 0

#### Scenario: Unknown email
- **WHEN** `vynilino user deactivate` is invoked for an email that does not exist
- **THEN** the command SHALL print "user not found" and exit with code 1

### Requirement: `user activate` subcommand
The system SHALL expose a `vynilino user activate --email <email>` command that re-enables a previously deactivated user.

#### Scenario: User activated successfully
- **WHEN** `vynilino user activate --email user@example.com` is invoked for an inactive user
- **THEN** the command SHALL mark the user active and print "user activated" and exit with code 0

#### Scenario: Already active user
- **WHEN** `vynilino user activate` is invoked for a user already active
- **THEN** the command SHALL print "user already active" and exit with code 0

### Requirement: `user change-password` subcommand
The system SHALL expose a `vynilino user change-password --email <email>` command that replaces a user's password hash. The new password SHALL be prompted interactively or read from stdin when `--password-stdin` is passed.

#### Scenario: Password changed with interactive prompt
- **WHEN** `vynilino user change-password --email user@example.com` is invoked and a valid new password is entered
- **THEN** the command SHALL update the password hash and print "password updated" and exit with code 0

#### Scenario: Password changed with stdin
- **WHEN** `vynilino user change-password --email user@example.com --password-stdin` is invoked with the new password on stdin
- **THEN** the command SHALL update the password hash and print "password updated" and exit with code 0

#### Scenario: Weak new password rejected
- **WHEN** `vynilino user change-password` is invoked with a password shorter than 12 characters
- **THEN** the command SHALL print "password too weak" and exit with code 1

### Requirement: DB path flag on all admin commands
All `user` subcommands SHALL accept a `--db` flag (default: value of `VYNILINO_DB_PATH` env var, fallback `./vynilino.db`) to specify the SQLite database file path.

#### Scenario: Custom DB path used
- **WHEN** `vynilino user list --db /data/prod.db` is invoked
- **THEN** the command SHALL open `/data/prod.db` and operate on that database

#### Scenario: Default DB path from env var
- **WHEN** `VYNILINO_DB_PATH=/data/prod.db` is set and `vynilino user list` is invoked without `--db`
- **THEN** the command SHALL open `/data/prod.db`
