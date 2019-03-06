-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Version exposing (Model, Msg(..), initial, update)

import Error
import Html exposing (..)
import Html.Attributes exposing (..)
import Http
import Json.Decode as D
import Json.Encode as E
import Url.Builder as B


url =
    { base = "http://localhost"
    , path = [ "version" ]
    }


type alias Model =
    { version : String }


initial : Model
initial =
    { version = "" }


type Msg
    = GetVersion
    | GotVersion (Result Http.Error String)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GetVersion ->
            ( model
            , Http.get
                { url = B.crossOrigin url.base url.path []
                , expect = Http.expectJson GotVersion (D.field "version" D.string)
                }
            )

        GotVersion result ->
            case result of
                Ok version ->
                    ( { model | version = version }, Cmd.none )

                Err error ->
                    ( { model | version = Error.handle error }, Cmd.none )
