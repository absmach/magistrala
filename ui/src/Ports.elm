-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


port module Ports exposing (websocketIn, websocketOut)

-- PORTS
-- JavaScript usage: app.ports.websocketIn.send(response);


port websocketIn : (String -> msg) -> Sub msg



-- JavaScript usage: app.ports.websocketOut.subscribe(handler);


port websocketOut : String -> Cmd msg
