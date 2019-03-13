-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Connection exposing (Model, Msg(..), initial, update, view)

import Bootstrap.Button as Button
import Bootstrap.Card as Card
import Bootstrap.Card.Block as Block
import Bootstrap.Form as Form
import Bootstrap.Form.Checkbox as Checkbox
import Bootstrap.Form.Input as Input
import Bootstrap.Grid as Grid
import Bootstrap.Table as Table
import Bootstrap.Text as Text
import Bootstrap.Utilities.Spacing as Spacing
import Channel
import Debug exposing (log)
import Error
import Helpers
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import HttpMF exposing (path)
import List.Extra
import Thing
import Url.Builder as B


type alias Model =
    { response : String
    , things : Thing.Model
    , channels : Channel.Model
    , checkedThingsIds : List String
    , checkedChannelsIds : List String
    }


initial : Model
initial =
    { response = ""
    , things = Thing.initial
    , channels = Channel.initial
    , checkedThingsIds = []
    , checkedChannelsIds = []
    }


type Msg
    = Connect
    | Disconnect
    | ThingMsg Thing.Msg
    | ChannelMsg Channel.Msg
    | GotResponse (Result Http.Error String)
    | CheckThing String
    | CheckChannel String


update : Msg -> Model -> String -> ( Model, Cmd Msg )
update msg model token =
    case msg of
        Connect ->
            if List.isEmpty model.checkedThingsIds || List.isEmpty model.checkedChannelsIds then
                ( model, Cmd.none )

            else
                ( { model | checkedThingsIds = [], checkedChannelsIds = [] }
                , Cmd.batch (connect model.checkedThingsIds model.checkedChannelsIds "PUT" token)
                )

        Disconnect ->
            if List.isEmpty model.checkedThingsIds || List.isEmpty model.checkedChannelsIds then
                ( model, Cmd.none )

            else
                ( { model | checkedThingsIds = [], checkedChannelsIds = [] }
                , Cmd.batch (connect model.checkedThingsIds model.checkedChannelsIds "DELETE" token)
                )

        GotResponse result ->
            case result of
                Ok statusCode ->
                    ( { model | response = statusCode }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        ThingMsg subMsg ->
            let
                ( updatedThing, thingCmd ) =
                    Thing.update subMsg model.things token
            in
            ( { model | things = updatedThing }, Cmd.map ThingMsg thingCmd )

        ChannelMsg subMsg ->
            let
                ( updatedChannel, channelCmd ) =
                    Channel.update subMsg model.channels token
            in
            ( { model | channels = updatedChannel }, Cmd.map ChannelMsg channelCmd )

        CheckThing id ->
            ( { model | checkedThingsIds = Helpers.checkEntity id model.checkedThingsIds }, Cmd.none )

        CheckChannel id ->
            ( { model | checkedChannelsIds = Helpers.checkEntity id model.checkedChannelsIds }, Cmd.none )



-- VIEW


view : Model -> Html Msg
view model =
    Grid.container []
        [ Grid.row []
            [ Grid.col []
                [ Card.config
                    []
                    |> Card.headerH3 [] [ text "Things" ]
                    |> Card.block []
                        [ Block.custom
                            (Table.table
                                { options = [ Table.striped, Table.hover, Table.small ]
                                , thead =
                                    Table.simpleThead
                                        [ Table.th [] [ text "Name" ]
                                        , Table.th [] [ text "ID" ]
                                        ]
                                , tbody = Table.tbody [] <| genThingRows model.checkedThingsIds model.things.things.list
                                }
                            )
                        ]
                    |> Card.view
                , Html.map ThingMsg (Helpers.genPagination model.things.things.total Thing.SubmitPage)
                ]
            , Grid.col []
                [ Card.config
                    []
                    |> Card.headerH3 [] [ text "Channels" ]
                    |> Card.block []
                        [ Block.custom
                            (Table.table
                                { options = [ Table.striped, Table.hover, Table.small ]
                                , thead =
                                    Table.simpleThead
                                        [ Table.th [] [ text "Name" ]
                                        , Table.th [] [ text "ID" ]
                                        ]
                                , tbody = Table.tbody [] <| genChannelRows model.checkedChannelsIds model.channels.channels.list
                                }
                            )
                        ]
                    |> Card.view
                , Html.map ChannelMsg (Helpers.genPagination model.channels.channels.total Channel.SubmitPage)
                ]
            ]
        , Grid.row []
            [ Grid.col []
                [ Form.form []
                    [ Button.button [ Button.success, Button.attrs [ Spacing.ml1 ], Button.onClick Connect ] [ text "Connect" ]
                    , Button.button [ Button.danger, Button.attrs [ Spacing.ml1 ], Button.onClick Disconnect ] [ text "Disconnect" ]
                    ]
                ]
            ]
        , Helpers.response model.response
        ]


genThingRows : List String -> List Thing.Thing -> List (Table.Row Msg)
genThingRows checkedThingsIds things =
    List.map
        (\thing ->
            Table.tr []
                [ Table.td [] [ input [ type_ "checkbox", onClick (CheckThing thing.id), checked (Helpers.isChecked thing.id checkedThingsIds) ] [], text (" " ++ Helpers.parseString thing.name) ]
                , Table.td [] [ text thing.id ]
                ]
        )
        things


genChannelRows : List String -> List Channel.Channel -> List (Table.Row Msg)
genChannelRows checkedChannelsIds channels =
    List.map
        (\channel ->
            Table.tr []
                [ Table.td [] [ input [ type_ "checkbox", onClick (CheckChannel channel.id), checked (Helpers.isChecked channel.id checkedChannelsIds) ] [], text (" " ++ Helpers.parseString channel.name) ]
                , Table.td [] [ text channel.id ]
                ]
        )
        channels



-- HTTP


connect : List String -> List String -> String -> String -> List (Cmd Msg)
connect checkedThingsIds checkedChannelsIds method token =
    List.foldr (++)
        []
        (List.map
            (\thingid ->
                List.map
                    (\channelid ->
                        HttpMF.request
                            (B.relative [ path.channels, channelid, path.things, thingid ] [])
                            method
                            token
                            Http.emptyBody
                            GotResponse
                    )
                    checkedChannelsIds
            )
            checkedThingsIds
        )
