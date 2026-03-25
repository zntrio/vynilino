/**
 * Thin GraphQL client. Authentication is handled via the HttpOnly
 * vynilino_access cookie that the browser sends automatically with every
 * same-origin request (THREAT-006 mitigation).
 */
export async function gql(query, variables = {}) {
  const res = await fetch('/graphql', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ query, variables }),
  })

  if (!res.ok) {
    throw new Error(`GraphQL HTTP error: ${res.status}`)
  }

  const json = await res.json()
  if (json.errors?.length) {
    throw Object.assign(new Error(json.errors[0].message), { graphqlErrors: json.errors })
  }
  return json.data
}

/** Search the Discogs database. Returns an array of DiscogsResult objects. */
export async function searchDiscogs(query, type = null) {
  const QUERY = /* graphql */`
    query SearchDiscogs($query: String!, $type: DiscogsSearchType) {
      searchDiscogs(query: $query, type: $type) {
        discogsId title artist year label format thumbUrl country
      }
    }
  `
  const data = await gql(QUERY, { query, type })
  return data.searchDiscogs
}

/** Subscribe to a GraphQL subscription over WebSocket (graphql-transport-ws). */
export function subscribe(query, variables = {}, onData, onError) {
  // The browser sends the vynilino_access HttpOnly cookie with the WebSocket
  // upgrade request automatically; no need to include it in the payload.
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(`${protocol}//${location.host}/graphql`, 'graphql-transport-ws')

  ws.addEventListener('open', () => {
    ws.send(JSON.stringify({ type: 'connection_init', payload: {} }))
  })

  let subId = null

  ws.addEventListener('message', (event) => {
    const msg = JSON.parse(event.data)
    switch (msg.type) {
      case 'connection_ack':
        subId = String(Date.now())
        ws.send(JSON.stringify({ id: subId, type: 'subscribe', payload: { query, variables } }))
        break
      case 'next':
        if (msg.payload?.data) onData(msg.payload.data)
        break
      case 'error':
        onError?.(msg.payload)
        break
      case 'complete':
        if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
          ws.close()
        }
        break
    }
  })

  ws.addEventListener('error', (e) => onError?.(e))
  ws.addEventListener('close', (e) => {
    if (e.code === 4401) onError?.({ code: 4401, reason: 'Unauthorized' })
  })

  return () => {
    if (subId && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ id: subId, type: 'complete' }))
    }
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      ws.close()
    }
  }
}
