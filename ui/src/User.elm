-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module User exposing (Model, Msg(..), initial, loggedIn, subscriptions, update, view)

import Bootstrap.Button as Button
import Bootstrap.Dropdown as Dropdown
import Bootstrap.Form as Form
import Bootstrap.Form.Input as Input
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Utilities.Spacing as Spacing
import Error
import Helpers
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import HttpMF exposing (baseURL, paths)
import Json.Decode as D
import Json.Encode as E
import Url.Builder as B


type alias Model =
    { email : String
    , password : String
    , token : String
    , response : String
    , dropState : Dropdown.State
    }


initial : Model
initial =
    { email = ""
    , password = ""
    , token = ""
    , response = ""
    , dropState = Dropdown.initialState
    }


type Msg
    = SubmitEmail String
    | SubmitPassword String
    | Create
    | Created (Result Http.Error String)
    | GetToken
    | GotToken (Result Http.Error String)
    | DropState Dropdown.State
    | LogOut


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        SubmitEmail email ->
            ( { model | email = email }, Cmd.none )

        SubmitPassword password ->
            ( { model | password = password }, Cmd.none )

        Create ->
            ( model
            , HttpMF.user
                model.email
                model.password
                (B.relative [ paths.users ] [])
                (encode (User model.email model.password))
                (HttpMF.expectStatus Created)
            )

        Created result ->
            case result of
                Ok statusCode ->
                    ( { model | response = statusCode }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        GetToken ->
            ( model
            , HttpMF.user
                model.email
                model.password
                (B.relative [ paths.tokens ] [])
                (encode (User model.email model.password))
                (HttpMF.expectRetrieve
                    GotToken
                    (D.field "token" D.string)
                )
            )

        GotToken result ->
            case result of
                Ok token ->
                    ( { model | token = token, response = "" }, Cmd.none )

                Err error ->
                    ( { model | token = "", response = Error.handle error }, Cmd.none )

        DropState state ->
            ( { model | dropState = state }, Cmd.none )

        LogOut ->
            ( { model | email = "", password = "", token = "", response = "" }, Cmd.none )



-- VIEW


view : Model -> Html Msg
view model =
    if loggedIn model then
        Grid.row []
            [ Grid.col [ Col.attrs [ align "right" ] ]
                [ Dropdown.dropdown
                    model.dropState
                    { options = []
                    , toggleMsg = DropState
                    , toggleButton =
                        Dropdown.toggle [ Button.warning ] [ text model.email ]
                    , items =
                        [ Dropdown.buttonItem [ onClick LogOut ] [ text "logout" ]
                        ]
                    }
                ]
            ]

    else
        Grid.container []
            [ Grid.row []
                [ Grid.col []
                    [ Form.form []
                        [ Form.group []
                            [ Form.label [ for "email" ] [ text "Email address" ]
                            , Input.email [ Input.id "email", Input.onInput SubmitEmail ]
                            ]
                        , Form.group []
                            [ Form.label [ for "pwd" ] [ text "Password" ]
                            , Input.password [ Input.id "pwd", Input.onInput SubmitPassword ]
                            ]
                        , Button.button [ Button.primary, Button.attrs [ Spacing.ml1 ], Button.onClick Create ] [ text "Register" ]
                        , Button.button [ Button.primary, Button.attrs [ Spacing.ml1 ], Button.onClick GetToken ] [ text "Log in" ]
                        ]
                    ]
                ]
            , Helpers.response model.response
            ]


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Dropdown.subscriptions model.dropState DropState ]



-- JSON


type alias User =
    { email : String
    , password : String
    }


encode : User -> E.Value
encode user =
    E.object
        [ ( "email", E.string user.email )
        , ( "password", E.string user.password )
        ]


decoder : D.Decoder User
decoder =
    D.map2 User
        (D.field "email" D.string)
        (D.field "password" D.string)



-- HTTP


loggedIn : Model -> Bool
loggedIn model =
    if String.length model.token > 0 then
        True

    else
        False
