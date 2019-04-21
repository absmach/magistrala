-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


port module Main exposing (Model, Msg(..), init, main, subscriptions, update, view)

import Bootstrap.Button as Button
import Bootstrap.ButtonGroup as ButtonGroup
import Bootstrap.CDN as CDN
import Bootstrap.Card as Card
import Bootstrap.Card.Block as Block
import Bootstrap.Form as Form
import Bootstrap.Form.Checkbox as Checkbox
import Bootstrap.Form.Fieldset as Fieldset
import Bootstrap.Form.Input as Input
import Bootstrap.Form.Radio as Radio
import Bootstrap.Form.Select as Select
import Bootstrap.Form.Textarea as Textarea
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Grid.Row as Row
import Bootstrap.Text as Text
import Bootstrap.Utilities.Spacing as Spacing
import Browser
import Browser.Navigation as Nav
import Channel
import Connection
import Debug exposing (log)
import Error
import Helpers exposing (faIcons, fontAwesome)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import Json.Decode exposing (Decoder, field, string)
import Json.Encode as Encode
import Message
import Thing
import Url
import Url.Parser as UrlParser exposing ((</>))
import User
import Version



-- MAIN


main : Program () Model Msg
main =
    Browser.application
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        , onUrlChange = UrlChanged
        , onUrlRequest = LinkClicked
        }



-- MODEL


type alias Model =
    { key : Nav.Key
    , user : User.Model
    , dashboard : Version.Model
    , channel : Channel.Model
    , thing : Thing.Model
    , connection : Connection.Model
    , message : Message.Model
    , view : String
    }


init : () -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init _ url key =
    ( Model key
        User.initial
        Version.initial
        Channel.initial
        Thing.initial
        Connection.initial
        Message.initial
        (parse url)
    , Cmd.none
    )



-- URL PARSER


type alias Route =
    ( String, Maybe String )


parse : Url.Url -> String
parse url =
    UrlParser.parse
        (UrlParser.map Tuple.pair (UrlParser.string </> UrlParser.fragment identity))
        url
        |> (\route ->
                case route of
                    Just r ->
                        Tuple.first r

                    Nothing ->
                        ""
           )


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | UserMsg User.Msg
    | VersionMsg Version.Msg
    | ChannelMsg Channel.Msg
    | ThingMsg Thing.Msg
    | ConnectionMsg Connection.Msg
    | MessageMsg Message.Msg
    | Version
    | Channels
    | Things
    | Connection
    | Messages



-- UPDATE


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        LinkClicked urlRequest ->
            case urlRequest of
                Browser.Internal url ->
                    ( model, Nav.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Cmd.none )

        UrlChanged url ->
            ( { model | view = parse url }
            , Cmd.none
            )

        UserMsg subMsg ->
            updateUser model subMsg

        VersionMsg subMsg ->
            updateVersion model subMsg

        ChannelMsg subMsg ->
            updateChannel model subMsg

        ThingMsg subMsg ->
            updateThing model subMsg

        ConnectionMsg subMsg ->
            updateConnection model subMsg

        MessageMsg subMsg ->
            updateMessage model subMsg

        Version ->
            ( { model | view = "dashboard" }, Cmd.none )

        Things ->
            ( { model | view = "things" }, Cmd.none )

        Channels ->
            ( { model | view = "channels" }, Cmd.none )

        Connection ->
            ( { model | view = "connection" }
            , Cmd.batch
                [ Tuple.second (updateConnection model (Connection.ThingMsg Thing.RetrieveThings))
                , Tuple.second (updateConnection model (Connection.ChannelMsg Channel.RetrieveChannels))
                ]
            )

        Messages ->
            updateMessage { model | view = "messages" } (Message.ThingMsg Thing.RetrieveThings)


updateUser : Model -> User.Msg -> ( Model, Cmd Msg )
updateUser model msg =
    let
        ( updatedUser, userCmd ) =
            User.update msg model.user
    in
    case msg of
        User.GotToken _ ->
            if String.length updatedUser.token > 0 then
                logIn { model | view = "dashboard" } updatedUser Version.GetVersion Thing.RetrieveThings Channel.RetrieveChannels

            else
                ( { model | user = updatedUser }, Cmd.map UserMsg userCmd )

        _ ->
            ( { model | user = updatedUser }, Cmd.map UserMsg userCmd )


logIn : Model -> User.Model -> Version.Msg -> Thing.Msg -> Channel.Msg -> ( Model, Cmd Msg )
logIn model user dashboardMsg thingMsg channelMsg =
    let
        ( updatedVersion, dashboardCmd ) =
            Version.update dashboardMsg model.dashboard

        ( updatedThing, thingCmd ) =
            Thing.update thingMsg model.thing user.token

        ( updatedChannel, channelCmd ) =
            Channel.update channelMsg model.channel user.token
    in
    ( { model | user = user }
    , Cmd.batch
        [ Cmd.map VersionMsg dashboardCmd
        , Cmd.map ThingMsg thingCmd
        , Cmd.map ChannelMsg channelCmd
        ]
    )


updateVersion : Model -> Version.Msg -> ( Model, Cmd Msg )
updateVersion model msg =
    let
        ( updatedVersion, dashboardCmd ) =
            Version.update msg model.dashboard
    in
    ( { model | dashboard = updatedVersion }, Cmd.map VersionMsg dashboardCmd )


updateThing : Model -> Thing.Msg -> ( Model, Cmd Msg )
updateThing model msg =
    let
        ( updatedThing, thingCmd ) =
            Thing.update msg model.thing model.user.token
    in
    ( { model | thing = updatedThing }, Cmd.map ThingMsg thingCmd )


updateChannel : Model -> Channel.Msg -> ( Model, Cmd Msg )
updateChannel model msg =
    let
        ( updatedChannel, channelCmd ) =
            Channel.update msg model.channel model.user.token
    in
    ( { model | channel = updatedChannel }, Cmd.map ChannelMsg channelCmd )


updateConnection : Model -> Connection.Msg -> ( Model, Cmd Msg )
updateConnection model msg =
    let
        ( updatedConnection, connectionCmd ) =
            Connection.update msg model.connection model.user.token
    in
    ( { model | connection = updatedConnection }, Cmd.map ConnectionMsg connectionCmd )


updateMessage : Model -> Message.Msg -> ( Model, Cmd Msg )
updateMessage model msg =
    let
        ( updatedMessage, messageCmd ) =
            Message.update msg model.message model.user.token
    in
    ( { model | message = updatedMessage }, Cmd.map MessageMsg messageCmd )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Sub.map UserMsg (User.subscriptions model.user)
        ]



-- VIEW


mfStylesheet : Html msg
mfStylesheet =
    node "link"
        [ rel "stylesheet"
        , href "./css/mainflux.css"
        ]
        []


view : Model -> Browser.Document Msg
view model =
    { title = "Mainflux"
    , body =
        let
            buttonAttrs =
                Button.attrs [ style "text-align" "left" ]

            loggedIn =
                User.loggedIn model.user

            menu =
                if loggedIn then
                    [ ul [ class "nav-pills flex-column nav" ]
                        [ menuItem "Dashboard" Version faIcons.dashboard (model.view == "dashboard")
                        , menuItem "Things" Things faIcons.things (model.view == "things")
                        , menuItem "Channels" Channels faIcons.channels (model.view == "channels")
                        , menuItem "Connection" Connection faIcons.connection (model.view == "connection")
                        , menuItem "Messages" Messages faIcons.messages (model.view == "messages")
                        ]
                    ]

                else
                    []

            header =
                if loggedIn then
                    Html.map UserMsg (User.view model.user)

                else
                    Grid.container [] []

            content =
                if loggedIn then
                    case model.view of
                        "dashboard" ->
                            dashboard model

                        "channels" ->
                            Html.map ChannelMsg (Channel.view model.channel)

                        "things" ->
                            Html.map ThingMsg (Thing.view model.thing)

                        "connection" ->
                            Html.map ConnectionMsg (Connection.view model.connection)

                        "messages" ->
                            Html.map MessageMsg (Message.view model.message)

                        _ ->
                            dashboard model

                else
                    Html.map UserMsg (User.view model.user)
        in
        [ Grid.containerFluid []
            [ CDN.stylesheet -- creates an inline style node with the Bootstrap CSS
            , mfStylesheet
            , fontAwesome
            , Grid.row [ Row.attrs [ style "height" "100vh" ] ]
                [ Grid.col
                    [ Col.attrs
                        [ style "background-color" "#113f67"
                        , style "padding" "0"
                        , style "color" "white"
                        ]
                    ]
                    [ Grid.row []
                        [ Grid.col
                            [ Col.attrs [] ]
                            [ h3 [ class "title" ] [ text "MAINFLUX" ] ]
                        ]
                    , Grid.row []
                        [ Grid.col
                            []
                            menu
                        ]
                    ]
                , Grid.col
                    [ Col.xs10
                    , Col.attrs []
                    ]
                    [ header
                    , Grid.row []
                        [ Grid.col
                            [ Col.attrs [] ]
                            [ content ]
                        ]
                    ]
                ]
            ]
        ]
    }


dashboard : Model -> Html Msg
dashboard model =
    Grid.container
        []
        [ Grid.row []
            [ Grid.col []
                [ Card.deck (cardList model)
                ]
            ]
        ]


cardList : Model -> List (Card.Config Msg)
cardList model =
    [ Card.config []
        |> Card.headerH3 [] [ div [ class "table_header" ] [ i [ style "margin-right" "15px", class faIcons.version ] [], text "Version" ] ]
        |> Card.block []
            [ Block.titleH4 [] [ text model.dashboard.version ] ]
    , Card.config []
        |> Card.headerH3 [] [ div [ class "table_header" ] [ i [ style "margin-right" "15px", class faIcons.things ] [], text "Things" ] ]
        |> Card.block []
            [ Block.titleH4 [] [ text (String.fromInt model.thing.things.total) ]
            , Block.custom <|
                Button.button [ Button.secondary, Button.onClick Things ] [ text "Manage things" ]
            ]
    , Card.config []
        |> Card.headerH3 [] [ div [ class "table_header" ] [ i [ style "margin-right" "15px", class faIcons.channels ] [], text "Channels" ] ]
        |> Card.block []
            [ Block.titleH4 [] [ text (String.fromInt model.channel.channels.total) ]
            , Block.custom <|
                Button.button [ Button.secondary, Button.onClick Channels ] [ text "Manage channels" ]
            ]
    ]


menuItem : String -> Msg -> String -> Bool -> Html Msg
menuItem name msg icon active =
    li [ class "nav-item", class "text-left" ] [ a [ onClick msg, classList [ ( "nav-link", True ), ( "active", active ) ] ] [ i [ class icon ] [], text name ] ]
