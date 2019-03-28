-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Channel exposing (Channel, Model, Msg(..), initial, update, view)

import Bootstrap.Button as Button
import Bootstrap.Card as Card
import Bootstrap.Card.Block as Block
import Bootstrap.Form as Form
import Bootstrap.Form.Input as Input
import Bootstrap.Form.InputGroup as InputGroup
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Modal as Modal
import Bootstrap.Table as Table
import Bootstrap.Utilities.Spacing as Spacing
import Dict
import Error
import Helpers exposing (faIcons)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import HttpMF exposing (paths)
import Json.Decode as D
import Json.Encode as E
import ModalMF
import Url.Builder as B


query =
    { offset = 0
    , limit = 10
    }


type alias Channel =
    { name : Maybe String
    , id : String
    , metadata : Maybe String
    }


emptyChannel =
    Channel (Just "") "" (Just "")


type alias Channels =
    { list : List Channel
    , total : Int
    }


type alias Model =
    { name : String
    , metadata : String
    , offset : Int
    , limit : Int
    , response : String
    , channels : Channels
    , channel : Channel
    , editMode : Bool
    , provisionModalVisibility : Modal.Visibility
    , editModalVisibility : Modal.Visibility
    }


initial : Model
initial =
    { name = ""
    , metadata = ""
    , offset = query.offset
    , limit = query.limit
    , response = ""
    , channels =
        { list = []
        , total = 0
        }
    , channel = emptyChannel
    , editMode = False
    , provisionModalVisibility = Modal.hidden
    , editModalVisibility = Modal.hidden
    }


type Msg
    = SubmitName String
    | SubmitMetadata String
    | ProvisionChannel
    | ProvisionedChannel (Result Http.Error String)
    | EditChannel
    | UpdateChannel
    | UpdatedChannel (Result Http.Error String)
    | RetrieveChannel String
    | RetrievedChannel (Result Http.Error Channel)
    | RetrieveChannels
    | RetrieveChannelsForThing String
    | RetrievedChannels (Result Http.Error Channels)
    | RemoveChannel String
    | RemovedChannel (Result Http.Error String)
    | SubmitPage Int
    | ShowEditModal Channel
    | CloseEditModal
    | ShowProvisionModal
    | ClosePorvisionModal


update : Msg -> Model -> String -> ( Model, Cmd Msg )
update msg model token =
    case msg of
        SubmitName name ->
            ( { model | name = name }, Cmd.none )

        SubmitPage page ->
            updateChannelList { model | offset = Helpers.pageToOffset page query.limit } token

        SubmitMetadata metadata ->
            ( { model | metadata = metadata }, Cmd.none )

        ProvisionChannel ->
            ( resetEdit model
            , HttpMF.provision
                (B.relative [ paths.channels ] [])
                token
                { emptyChannel
                    | name = Just model.name
                    , metadata = Just model.metadata
                }
                channelEncoder
                ProvisionedChannel
                "/channels/"
            )

        ProvisionedChannel result ->
            case result of
                Ok channelid ->
                    updateChannelList
                        { model
                            | channel = { emptyChannel | id = channelid }
                            , provisionModalVisibility = Modal.hidden
                            , editModalVisibility = Modal.shown
                        }
                        token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        EditChannel ->
            ( { model
                | editMode = True
                , name = Helpers.parseString model.channel.name
                , metadata = Helpers.parseString model.channel.metadata
              }
            , Cmd.none
            )

        UpdateChannel ->
            ( resetEdit { model | editMode = False }
            , HttpMF.update
                (B.relative [ paths.channels, model.channel.id ] [])
                token
                { emptyChannel
                    | name = Just model.name
                    , metadata = Just model.metadata
                }
                channelEncoder
                UpdatedChannel
            )

        UpdatedChannel result ->
            case result of
                Ok statusCode ->
                    updateChannelList (resetEdit { model | response = statusCode }) token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RetrieveChannel channelid ->
            ( model
            , HttpMF.retrieve
                (B.relative [ paths.channels, channelid ] [])
                token
                RetrievedChannel
                channelDecoder
            )

        RetrievedChannel result ->
            case result of
                Ok channel ->
                    ( { model | channel = channel }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RetrieveChannels ->
            ( model
            , HttpMF.retrieve
                (B.relative [ paths.channels ] (Helpers.buildQueryParamList model.offset model.limit))
                token
                RetrievedChannels
                channelsDecoder
            )

        RetrieveChannelsForThing thingid ->
            ( model
            , HttpMF.retrieve
                (B.relative [ paths.things, thingid, paths.channels ] (Helpers.buildQueryParamList model.offset model.limit))
                token
                RetrievedChannels
                channelsDecoder
            )

        RetrievedChannels result ->
            case result of
                Ok channels ->
                    ( { model | channels = channels }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RemoveChannel id ->
            ( resetEdit model
            , HttpMF.remove
                (B.relative [ paths.channels, id ] [])
                token
                RemovedChannel
            )

        RemovedChannel result ->
            case result of
                Ok statusCode ->
                    updateChannelList
                        { model
                            | response = statusCode
                            , offset = Helpers.validateOffset model.offset model.channels.total query.limit
                            , editModalVisibility = Modal.hidden
                        }
                        token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        ShowEditModal channel ->
            ( { model
                | editModalVisibility = Modal.shown
                , channel = channel
                , editMode = False
              }
            , Cmd.none
            )

        CloseEditModal ->
            ( resetEdit { model | editModalVisibility = Modal.hidden }, Cmd.none )

        ShowProvisionModal ->
            ( { model | provisionModalVisibility = Modal.shown }
            , Cmd.none
            )

        ClosePorvisionModal ->
            ( resetEdit { model | provisionModalVisibility = Modal.hidden }, Cmd.none )



-- VIEW


view : Model -> Html Msg
view model =
    Grid.container []
        ([ Grid.row []
            [ Grid.col [ Col.attrs [ align "right" ] ]
                [ Button.button [ Button.outlinePrimary, Button.attrs [ Spacing.ml1, align "right" ], Button.onClick ShowProvisionModal ] [ text "ADD" ]
                ]
            ]
         , Grid.row []
            [ Grid.col []
                [ Card.config []
                    |> Card.headerH3 [] [ text "Channels" ]
                    |> Card.block []
                        [ Block.custom
                            (Table.table
                                { options = [ Table.striped, Table.hover, Table.small ]
                                , thead = genTableHeader
                                , tbody = genTableBody model
                                }
                            )
                        ]
                    |> Card.view
                ]
            ]
         , provisionModal model
         , editModal model
         ]
            |> Helpers.appendIf (model.channels.total > model.limit)
                (Helpers.genPagination model.channels.total (Helpers.offsetToPage model.offset model.limit) SubmitPage)
        )



-- Channels table


genTableHeader : Table.THead Msg
genTableHeader =
    Table.simpleThead
        [ Table.th [] [ text "Name" ]
        , Table.th [] [ text "ID" ]
        ]


genTableBody : Model -> Table.TBody Msg
genTableBody model =
    Table.tbody []
        (List.map
            (\channel ->
                Table.tr [ Table.rowAttr (onClick (ShowEditModal channel)) ]
                    [ Table.td [] [ text (Helpers.parseString channel.name) ]
                    , Table.td [] [ text channel.id ]
                    ]
            )
            model.channels.list
        )



-- Provision modal


provisionModal : Model -> Html Msg
provisionModal model =
    Modal.config ClosePorvisionModal
        |> Modal.large
        |> Modal.hideOnBackdropClick True
        |> Modal.h4 [] [ text "Add channel" ]
        |> provisionModalBody model
        |> Modal.view model.provisionModalVisibility


provisionModalBody : Model -> (Modal.Config Msg -> Modal.Config Msg)
provisionModalBody model =
    Modal.body []
        [ Grid.container []
            [ ModalMF.modalForm
                [ ModalMF.FormRecord "name" SubmitName model.name model.name
                , ModalMF.FormRecord "metadata" SubmitMetadata model.metadata model.metadata
                ]
            , ModalMF.provisionModalButtons ProvisionChannel ClosePorvisionModal
            ]
        ]



-- Edit modal


editModal : Model -> Html Msg
editModal model =
    Modal.config CloseEditModal
        |> Modal.large
        |> Modal.hideOnBackdropClick True
        |> Modal.h4 [] [ text (Helpers.parseString model.channel.name) ]
        |> editModalBody model
        |> Modal.view model.editModalVisibility


editModalBody : Model -> (Modal.Config Msg -> Modal.Config Msg)
editModalBody model =
    Modal.body []
        [ Grid.container []
            [ Grid.row []
                [ Grid.col []
                    [ editModalForm model
                    , ModalMF.modalDiv [ ( "id", model.channel.id ) ]
                    ]
                ]
            , ModalMF.editModalButtons model.editMode UpdateChannel EditChannel (ShowEditModal model.channel) (RemoveChannel model.channel.id) CloseEditModal
            ]
        ]


editModalForm : Model -> Html Msg
editModalForm model =
    if model.editMode then
        ModalMF.modalForm
            [ ModalMF.FormRecord "name" SubmitName (Helpers.parseString model.channel.name) model.name
            , ModalMF.FormRecord "metadata" SubmitMetadata (Helpers.parseString model.channel.metadata) model.metadata
            ]

    else
        ModalMF.modalDiv [ ( "name", Helpers.parseString model.channel.name ), ( "metadata", Helpers.parseString model.channel.metadata ) ]



-- JSON


channelDecoder : D.Decoder Channel
channelDecoder =
    D.map3 Channel
        (D.maybe (D.field "name" D.string))
        (D.field "id" D.string)
        (D.maybe (D.field "metadata" D.string))


channelsDecoder : D.Decoder Channels
channelsDecoder =
    D.map2 Channels
        (D.field "channels" (D.list channelDecoder))
        (D.field "total" D.int)


channelEncoder : Channel -> E.Value
channelEncoder channel =
    E.object
        [ ( "name", E.string (Helpers.parseString channel.name) )
        , ( "metadata", E.string (Helpers.parseString channel.metadata) )
        ]



-- HELPERS


resetEdit : Model -> Model
resetEdit model =
    { model | name = "", metadata = "" }


updateChannelList : Model -> String -> ( Model, Cmd Msg )
updateChannelList model token =
    ( model
    , Cmd.batch
        [ HttpMF.retrieve
            (B.relative [ paths.channels ] (Helpers.buildQueryParamList model.offset model.limit))
            token
            RetrievedChannels
            channelsDecoder
        , HttpMF.retrieve
            (B.relative [ paths.channels, model.channel.id ] [])
            token
            RetrievedChannel
            channelDecoder
        ]
    )


updateChannelListForThing : Model -> String -> String -> ( Model, Cmd Msg )
updateChannelListForThing model token thingid =
    ( model
    , HttpMF.retrieve
        (B.relative [ paths.things, thingid, paths.channels ] (Helpers.buildQueryParamList model.offset model.limit))
        token
        RetrievedChannels
        channelsDecoder
    )
