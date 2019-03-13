-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module HttpMF exposing (expectID, expectRetrieve, expectStatus, path, provision, remove, request, retrieve, update, url)

import Dict
import Helpers
import Http
import Json.Decode as D
import Json.Encode as E
import Url.Builder as B


url =
    { base = "http://localhost"
    }


path =
    { users = "users"
    , tokens = "tokens"
    , things = "things"
    , channels = "channels"
    , messages = "messages"
    , version = "version"
    }



-- EXPECT


expectStatus : (Result Http.Error String -> msg) -> Http.Expect msg
expectStatus toMsg =
    Http.expectStringResponse toMsg <|
        \resp ->
            case resp of
                Http.BadUrl_ u ->
                    Err (Http.BadUrl u)

                Http.Timeout_ ->
                    Err Http.Timeout

                Http.NetworkError_ ->
                    Err Http.NetworkError

                Http.BadStatus_ metadata body ->
                    Err (Http.BadStatus metadata.statusCode)

                Http.GoodStatus_ metadata _ ->
                    Ok (String.fromInt metadata.statusCode)


expectID : (Result Http.Error String -> msg) -> String -> Http.Expect msg
expectID toMsg prefix =
    Http.expectStringResponse toMsg <|
        \resp ->
            case resp of
                Http.BadUrl_ u ->
                    Err (Http.BadUrl u)

                Http.Timeout_ ->
                    Err Http.Timeout

                Http.NetworkError_ ->
                    Err Http.NetworkError

                Http.BadStatus_ metadata body ->
                    Err (Http.BadStatus metadata.statusCode)

                Http.GoodStatus_ metadata body ->
                    Ok <|
                        String.dropLeft (String.length prefix) <|
                            Helpers.parseString (Dict.get "location" metadata.headers)


expectRetrieve : (Result Http.Error a -> msg) -> D.Decoder a -> Http.Expect msg
expectRetrieve toMsg decoder =
    Http.expectStringResponse toMsg <|
        \resp ->
            case resp of
                Http.BadUrl_ u ->
                    Err (Http.BadUrl u)

                Http.Timeout_ ->
                    Err Http.Timeout

                Http.NetworkError_ ->
                    Err Http.NetworkError

                Http.BadStatus_ metadata body ->
                    Err (Http.BadStatus metadata.statusCode)

                Http.GoodStatus_ metadata body ->
                    case D.decodeString decoder body of
                        Ok value ->
                            Ok value

                        Err err ->
                            Err (Http.BadBody (D.errorToString err))



-- REQUEST


request : String -> String -> String -> Http.Body -> (Result Http.Error String -> msg) -> Cmd msg
request u method token b msg =
    Http.request
        { method = method
        , headers = [ Http.header "Authorization" token ]
        , url = u
        , body = b
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }


retrieve : String -> String -> (Result Http.Error a -> msg) -> D.Decoder a -> Cmd msg
retrieve u token msg decoder =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" token ]
        , url = u
        , body = Http.emptyBody
        , expect = expectRetrieve msg decoder
        , timeout = Nothing
        , tracker = Nothing
        }


provision : String -> String -> entity -> (entity -> E.Value) -> (Result Http.Error String -> msg) -> String -> Cmd msg
provision u token e encoder msg prefix =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" token ]
        , url = u
        , body =
            encoder e
                |> Http.jsonBody
        , expect = expectID msg prefix
        , timeout = Nothing
        , tracker = Nothing
        }


update : String -> String -> entity -> (entity -> E.Value) -> (Result Http.Error String -> msg) -> Cmd msg
update u token e encoder msg =
    Http.request
        { method = "PUT"
        , headers = [ Http.header "Authorization" token ]
        , url = u
        , body =
            encoder e
                |> Http.jsonBody
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }


remove : String -> String -> (Result Http.Error String -> msg) -> Cmd msg
remove u token msg =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" token ]
        , url = u
        , body = Http.emptyBody
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }
