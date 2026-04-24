import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { GameService } from '@@gen/dominion/v1/game_pb'

const transport = createConnectTransport({ baseUrl: '/' })

export const client = createClient(GameService, transport)
