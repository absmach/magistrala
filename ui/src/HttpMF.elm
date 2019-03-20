-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module HttpMF exposing (baseURL, expectID, expectRetrieve, expectStatus, paths, provision, remove, request, retrieve, update, user, version)

import Dict
import Env exposing (env)
import Helpers
import Http
import Json.Decode as D
import Json.Encode as E
import Url.Builder as B


baseURL =
    env.url


paths =
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


version : String -> (Result Http.Error String -> msg) -> D.Decoder String -> Cmd msg
version path msg decoder =
    Http.get
        { url = baseURL ++ path
        , expect = Http.expectJson msg decoder
        }


user : String -> String -> String -> E.Value -> Http.Expect msg -> Cmd msg
user email password u value expect =
    Http.post
        { url = baseURL ++ u
        , body =
            value |> Http.jsonBody
        , expect = expect
        }


request : String -> String -> String -> Http.Body -> (Result Http.Error String -> msg) -> Cmd msg
request path method token b msg =
    Http.request
        { method = method
        , headers = [ Http.header "Authorization" token ]
        , url = baseURL ++ path
        , body = b
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }


retrieve : String -> String -> (Result Http.Error a -> msg) -> D.Decoder a -> Cmd msg
retrieve path token msg decoder =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" token ]
        , url = baseURL ++ path
        , body = Http.emptyBody
        , expect = expectRetrieve msg decoder
        , timeout = Nothing
        , tracker = Nothing
        }


provision : String -> String -> entity -> (entity -> E.Value) -> (Result Http.Error String -> msg) -> String -> Cmd msg
provision path token e encoder msg prefix =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" token ]
        , url = baseURL ++ path
        , body =
            encoder e
                |> Http.jsonBody
        , expect = expectID msg prefix
        , timeout = Nothing
        , tracker = Nothing
        }


update : String -> String -> entity -> (entity -> E.Value) -> (Result Http.Error String -> msg) -> Cmd msg
update path token e encoder msg =
    Http.request
        { method = "PUT"
        , headers = [ Http.header "Authorization" token ]
        , url = baseURL ++ path
        , body =
            encoder e
                |> Http.jsonBody
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }


remove : String -> String -> (Result Http.Error String -> msg) -> Cmd msg
remove path token msg =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" token ]
        , url = baseURL ++ path
        , body = Http.emptyBody
        , expect = expectStatus msg
        , timeout = Nothing
        , tracker = Nothing
        }
