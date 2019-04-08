-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Error exposing (handle)

import Debug
import Http


handle : Http.Error -> String
handle error =
    case error of
        Http.BadUrl url ->
            "Bad URL: " ++ url

        Http.Timeout ->
            "Timeout"

        Http.NetworkError ->
            "Network error"

        Http.BadStatus code ->
            "Bad status: " ++ String.fromInt code

        Http.BadBody err ->
            let
                _ =
                    Debug.log "Bad body error: " err
            in
            "Invalid response body"
