-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


port module Message exposing (Model, Msg(..), initial, subscriptions, update, view)

import Bootstrap.Button as Button
import Bootstrap.Card as Card
import Bootstrap.Card.Block as Block
import Bootstrap.Form as Form
import Bootstrap.Form.Checkbox as Checkbox
import Bootstrap.Form.Input as Input
import Bootstrap.Form.Radio as Radio
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Table as Table
import Bootstrap.Utilities.Spacing as Spacing
import Channel
import Debug exposing (log)
import Error
import Helpers exposing (faIcons, fontAwesome)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import HttpMF exposing (paths)
import Json.Decode as D
import Json.Encode as E
import List.Extra
import Ports exposing (..)
import Thing
import Url.Builder as B


type alias CheckedChannel =
    { id : String
    , ws : Int
    }


type alias Model =
    { message : String
    , thingkey : String
    , response : String
    , things : Thing.Model
    , channels : Channel.Model
    , thingid : String
    , checkedChannelsIds : List String
    , checkedChannelsIdsWs : List String
    , websocketIn : List String
    }


initial : Model
initial =
    { message = ""
    , thingkey = ""
    , response = ""
    , things = Thing.initial
    , channels = Channel.initial
    , thingid = ""
    , checkedChannelsIds = []
    , checkedChannelsIdsWs = []
    , websocketIn = []
    }


type Msg
    = SubmitMessage String
    | SendMessage
    | Listen
    | Stop
    | WebsocketIn String
    | RetrievedWebsockets E.Value
    | SentMessage (Result Http.Error String)
    | ThingMsg Thing.Msg
    | ChannelMsg Channel.Msg
    | SelectThing String String Channel.Msg
    | CheckChannel String


resetSent : Model -> Model
resetSent model =
    { model | message = "", thingkey = "", response = "", thingid = "" }


update : Msg -> Model -> String -> ( Model, Cmd Msg )
update msg model token =
    case msg of
        SubmitMessage message ->
            ( { model | message = message }, Cmd.none )

        SendMessage ->
            ( model
            , Cmd.batch
                (List.map
                    (\channelid -> send channelid model.thingkey model.message)
                    model.checkedChannelsIds
                )
            )

        Listen ->
            if List.isEmpty model.checkedChannelsIds then
                ( model, Cmd.none )

            else
                ( model, Cmd.batch <| ws connectWebsocket model )

        Stop ->
            if List.isEmpty model.checkedChannelsIds then
                ( model, Cmd.none )

            else
                ( model, Cmd.batch <| ws disconnectWebsocket model )

        SentMessage result ->
            case result of
                Ok statusCode ->
                    ( { model | response = statusCode }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        WebsocketIn data ->
            ( { model | websocketIn = data :: model.websocketIn }, Cmd.none )

        RetrievedWebsockets wssList ->
            case D.decodeValue websocketsQueryDecoder wssList of
                Ok wssL ->
                    if List.isEmpty wssL then
                        ( model, Cmd.none )

                    else
                        let
                            l =
                                List.map
                                    (\wss ->
                                        channelIdFromUrl wss.url
                                    )
                                    wssL
                        in
                        ( { model | checkedChannelsIdsWs = l }, Cmd.none )

                Err _ ->
                    ( model, Cmd.none )

        ThingMsg subMsg ->
            updateThing model subMsg token

        ChannelMsg subMsg ->
            updateChannel model subMsg token

        SelectThing thingid thingkey channelMsg ->
            updateChannel { model | thingid = thingid, thingkey = thingkey, checkedChannelsIds = [], checkedChannelsIdsWs = [] } (Channel.RetrieveChannelsForThing thingid) token

        CheckChannel id ->
            ( { model | checkedChannelsIds = Helpers.checkEntity id model.checkedChannelsIds }, Cmd.none )


retrieveWebsocketsForThing : List Channel.Channel -> String -> Cmd Msg
retrieveWebsocketsForThing channels thingkey =
    let
        wssList =
            List.map
                (\channel ->
                    Websocket channel.id thingkey ""
                )
                channels
    in
    queryWebsockets (websocketsEncoder wssList)


updateThing : Model -> Thing.Msg -> String -> ( Model, Cmd Msg )
updateThing model msg token =
    let
        ( updatedThing, thingCmd ) =
            Thing.update msg model.things token
    in
    ( { model | things = updatedThing }
    , Cmd.map ThingMsg thingCmd
    )


updateChannel : Model -> Channel.Msg -> String -> ( Model, Cmd Msg )
updateChannel model msg token =
    let
        ( updatedChannel, channelCmd ) =
            Channel.update msg model.channels token

        checkedChannels =
            updatedChannel.channels.list
    in
    ( { model | channels = updatedChannel }
    , Cmd.map ChannelMsg channelCmd
    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ websocketIn WebsocketIn
        , retrieveWebsockets RetrievedWebsockets
        ]



-- VIEW


view : Model -> Html Msg
view model =
    Grid.container []
        [ Grid.row []
            [ Grid.col []
                (Helpers.appendIf (model.things.things.total > model.things.limit)
                    [ Helpers.genCardConfig faIcons.things "Things" (genThingRows model) ]
                    (Html.map ThingMsg (Helpers.genPagination model.things.things.total (Helpers.offsetToPage model.things.offset model.things.limit) Thing.SubmitPage))
                )
            , Grid.col []
                (Helpers.appendIf (model.channels.channels.total > model.channels.limit)
                    [ Helpers.genCardConfig faIcons.channels "Channels" (genChannelRows model) ]
                    (Html.map ChannelMsg (Helpers.genPagination model.channels.channels.total (Helpers.offsetToPage model.channels.offset model.channels.limit) Channel.SubmitPage))
                )
            ]
        , Grid.row []
            [ Grid.col []
                [ Card.config []
                    |> Card.headerH3 []
                        [ Grid.row []
                            [ Grid.col []
                                [ div [ class "table_header" ]
                                    [ i [ style "margin-right" "15px", class faIcons.send ] []
                                    , text "HTTP"
                                    ]
                                ]
                            , Grid.col [ Col.attrs [ align "right" ] ]
                                [ Form.group []
                                    [ Button.button [ Button.secondary, Button.attrs [ Spacing.ml1 ], Button.onClick SendMessage ] [ text "Send" ]
                                    ]
                                ]
                            ]
                        ]
                    |> Card.block []
                        [ Block.custom
                            (Grid.row []
                                [ Grid.col [] [ Input.text [ Input.id "message", Input.onInput SubmitMessage ] ]
                                ]
                            )
                        ]
                    |> Card.view
                ]
            , Grid.col []
                [ Card.config []
                    |> Card.headerH3 []
                        [ Grid.row []
                            [ Grid.col []
                                [ div [ class "table_header" ]
                                    [ i [ style "margin-right" "15px", class faIcons.receive ] []
                                    , text "WS"
                                    ]
                                ]
                            , Grid.col [ Col.attrs [ align "right" ] ]
                                [ Form.form []
                                    [ Form.group []
                                        [ Button.button [ Button.secondary, Button.attrs [ Spacing.ml1 ], Button.onClick Listen ] [ text "Listen" ]
                                        , Button.button [ Button.secondary, Button.attrs [ Spacing.ml1 ], Button.onClick Stop ] [ text "Stop" ]
                                        ]
                                    ]
                                ]
                            ]
                        ]
                    |> Card.block []
                        [ Block.custom
                            (Helpers.genOrderedList model.websocketIn)
                        ]
                    |> Card.view
                ]
            ]
        , Helpers.response model.response
        ]


genThingRows : Model -> List (Table.Row Msg)
genThingRows model =
    List.map
        (\thing ->
            Table.tr []
                [ Table.td [] [ label [] [ text (Helpers.parseString thing.name) ] ]
                , Table.td [] [ text thing.id ]
                , Table.td [] [ input [ type_ "radio", onClick (SelectThing thing.id thing.key (Channel.RetrieveChannelsForThing thing.id)), name "things" ] [] ]
                ]
        )
        model.things.things.list


genChannelRows : Model -> List (Table.Row Msg)
genChannelRows model =
    List.map
        (\channel ->
            Table.tr []
                [ Table.td [] [ text (" " ++ Helpers.parseString channel.name) ]
                , Table.td [] [ text (channel.id ++ isInList channel.id model.checkedChannelsIdsWs) ]
                , Table.td [] [ input [ type_ "checkbox", onClick (CheckChannel channel.id), checked (Helpers.isChecked channel.id model.checkedChannelsIds) ] [] ]
                ]
        )
        model.channels.channels.list


isInList : String -> List String -> String
isInList id idList =
    if List.member id idList then
        " *WS*"

    else
        ""



-- HTTP


send : String -> String -> String -> Cmd Msg
send channelid thingkey message =
    HttpMF.request
        (B.relative [ "http", paths.channels, channelid, paths.messages ] [])
        "POST"
        thingkey
        (Http.stringBody "application/json" message)
        SentMessage


ws : (E.Value -> Cmd Msg) -> Model -> List (Cmd Msg)
ws command model =
    List.map
        (\channelid ->
            command <| websocketEncoder (Websocket channelid model.thingkey "")
        )
        model.checkedChannelsIds
