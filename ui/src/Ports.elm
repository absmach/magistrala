-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


port module Ports exposing (Websocket, WebsocketQuery, channelIdFromUrl, connectWebsocket, disconnectWebsocket, queryWebsocket, queryWebsockets, retrieveWebsocket, retrieveWebsockets, websocketEncoder, websocketIn, websocketOut, websocketQueryDecoder, websocketsEncoder, websocketsQueryDecoder)

import Json.Decode as D
import Json.Encode as E


port connectWebsocket : E.Value -> Cmd msg


port disconnectWebsocket : E.Value -> Cmd msg


port websocketIn : (String -> msg) -> Sub msg


port websocketOut : E.Value -> Cmd msg


port queryWebsocket : E.Value -> Cmd msg


port retrieveWebsocket : (E.Value -> msg) -> Sub msg


port queryWebsockets : E.Value -> Cmd msg


port retrieveWebsockets : (E.Value -> msg) -> Sub msg



-- JSON


type alias WebsocketQuery =
    { url : String
    , readyState : Int
    }


type alias Websocket =
    { channelid : String
    , thingkey : String
    , message : String
    }


websocketQueryDecoder : D.Decoder WebsocketQuery
websocketQueryDecoder =
    D.map2 WebsocketQuery
        (D.field "url" D.string)
        (D.field "readyState" D.int)


websocketsQueryDecoder : D.Decoder (List WebsocketQuery)
websocketsQueryDecoder =
    D.list websocketQueryDecoder


websocketEncoder : Websocket -> E.Value
websocketEncoder ws =
    E.object
        [ ( "channelid", E.string ws.channelid )
        , ( "thingkey", E.string ws.thingkey )
        , ( "message", E.string ws.message )
        ]


websocketsEncoder : List Websocket -> E.Value
websocketsEncoder wss =
    E.list websocketEncoder wss


channelIdFromUrl : String -> String
channelIdFromUrl url =
    let
        start =
            String.length "wss://localhost/ws/channels/"

        end =
            String.length "wss://localhost/ws/channels/"
                + String.length "0522c54b-5b00-4aab-a2b0-6e3e54320995"
    in
    String.slice start end url
