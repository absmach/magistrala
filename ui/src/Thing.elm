-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Thing exposing (Model, Msg(..), Thing, initial, update, view)

import Bootstrap.Button as Button
import Bootstrap.ButtonGroup as ButtonGroup
import Bootstrap.Card as Card
import Bootstrap.Card.Block as Block
import Bootstrap.Dropdown as Dropdown
import Bootstrap.Form as Form
import Bootstrap.Form.Input as Input
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Grid.Row as Row
import Bootstrap.Modal as Modal
import Bootstrap.Table as Table
import Bootstrap.Utilities.Spacing as Spacing
import Debug exposing (log)
import Dict
import Error
import Helpers exposing (faIcons, fontAwesome)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (onClick)
import Http
import HttpMF exposing (paths)
import Json.Decode as D
import Json.Encode as E
import JsonMF exposing (..)
import ModalMF
import Url.Builder as B


query =
    { offset = 0
    , limit = 10
    }


type alias Thing =
    { name : Maybe String
    , id : String
    , key : String
    , metadata : Maybe JsonValue
    }


emptyThing =
    Thing (Just "") "" "" (Just ValueNull)


type alias Things =
    { list : List Thing
    , total : Int
    }


type alias Model =
    { name : String
    , metadata : String
    , offset : Int
    , limit : Int
    , response : String
    , things : Things
    , thing : Thing
    , location : String
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
    , things =
        { list = []
        , total = 0
        }
    , thing = emptyThing
    , location = ""
    , editMode = False
    , provisionModalVisibility = Modal.hidden
    , editModalVisibility = Modal.hidden
    }


type Msg
    = SubmitName String
    | SubmitMetadata String
    | ProvisionThing
    | ProvisionedThing (Result Http.Error String)
    | EditThing
    | UpdateThing
    | UpdatedThing (Result Http.Error String)
    | RetrieveThing String
    | RetrievedThing (Result Http.Error Thing)
    | RetrieveThings
    | RetrievedThings (Result Http.Error Things)
    | RemoveThing String
    | RemovedThing (Result Http.Error String)
    | SubmitPage Int
    | ClosePorvisionModal
    | CloseEditModal
    | ShowProvisionModal
    | ShowEditModal Thing


update : Msg -> Model -> String -> ( Model, Cmd Msg )
update msg model token =
    case msg of
        SubmitName name ->
            ( { model | name = name }, Cmd.none )

        SubmitMetadata metadata ->
            ( { model | metadata = metadata }, Cmd.none )

        SubmitPage page ->
            updateThingList { model | offset = Helpers.pageToOffset page query.limit } token

        ProvisionThing ->
            ( resetEdit model
            , HttpMF.provision
                (B.relative [ paths.things ] [])
                token
                { emptyThing
                    | name = Just model.name
                    , metadata = stringToMaybeJsonValue model.metadata
                }
                thingEncoder
                ProvisionedThing
                "/things/"
            )

        ProvisionedThing result ->
            case result of
                Ok thingid ->
                    updateThingList
                        { model
                            | thing = { emptyThing | id = thingid }
                            , provisionModalVisibility = Modal.hidden
                            , editModalVisibility = Modal.shown
                        }
                        token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        EditThing ->
            ( { model
                | editMode = True
                , name = Helpers.parseString model.thing.name
                , metadata = maybeJsonValueToString model.thing.metadata
              }
            , Cmd.none
            )

        UpdateThing ->
            ( resetEdit { model | editMode = False }
            , HttpMF.update
                (B.relative [ paths.things, model.thing.id ] [])
                token
                { emptyThing
                    | name = Just model.name
                    , metadata =
                        case stringToJsonValue model.metadata of
                            Ok jsonValue ->
                                Just jsonValue

                            Err err ->
                                model.thing.metadata
                }
                thingEncoder
                UpdatedThing
            )

        UpdatedThing result ->
            case result of
                Ok statusCode ->
                    updateThingList (resetEdit { model | response = statusCode }) token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RetrieveThing thingid ->
            ( model
            , HttpMF.retrieve
                (B.relative [ paths.things, thingid ] [])
                token
                RetrievedThing
                thingDecoder
            )

        RetrievedThing result ->
            case result of
                Ok thing ->
                    ( { model | thing = thing }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RetrieveThings ->
            ( model
            , HttpMF.retrieve
                (B.relative [ paths.things ] (Helpers.buildQueryParamList model.offset model.limit))
                token
                RetrievedThings
                thingsDecoder
            )

        RetrievedThings result ->
            case result of
                Ok things ->
                    ( { model | things = things }, Cmd.none )

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        RemoveThing id ->
            ( model
            , HttpMF.remove
                (B.relative [ paths.things, id ] [])
                token
                RemovedThing
            )

        RemovedThing result ->
            case result of
                Ok statusCode ->
                    updateThingList
                        { model
                            | response = statusCode
                            , offset = Helpers.validateOffset model.offset model.things.total query.limit
                            , editModalVisibility = Modal.hidden
                        }
                        token

                Err error ->
                    ( { model | response = Error.handle error }, Cmd.none )

        ClosePorvisionModal ->
            ( resetEdit { model | provisionModalVisibility = Modal.hidden }, Cmd.none )

        CloseEditModal ->
            ( resetEdit { model | editModalVisibility = Modal.hidden }, Cmd.none )

        ShowProvisionModal ->
            ( { model | provisionModalVisibility = Modal.shown }
            , Cmd.none
            )

        ShowEditModal thing ->
            ( { model
                | editModalVisibility = Modal.shown
                , thing = thing
                , editMode = False
              }
            , Cmd.none
            )



-- VIEW


view : Model -> Html Msg
view model =
    Grid.container []
        (Helpers.appendIf (model.things.total > model.limit)
            [ genTable model, provisionModal model, editModal model ]
            (Helpers.genPagination model.things.total (Helpers.offsetToPage model.offset model.limit) SubmitPage)
        )



-- Things table


genTable : Model -> Html Msg
genTable model =
    Grid.row []
        [ Grid.col []
            [ Card.config []
                |> Card.header []
                    [ Grid.row []
                        [ Grid.col [ Col.attrs [ align "left" ] ]
                            [ h3 [] [ div [ class "table_header" ] [ i [ style "margin-right" "15px", class faIcons.things ] [], text "Things" ] ]
                            ]
                        , Grid.col [ Col.attrs [ align "right" ] ]
                            [ Button.button [ Button.secondary, Button.attrs [ align "right" ], Button.onClick ShowProvisionModal ] [ text "ADD" ]
                            ]
                        ]
                    ]
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
            (\thing ->
                Table.tr [ Table.rowAttr (onClick (ShowEditModal thing)) ]
                    [ Table.td [] [ text (Helpers.parseString thing.name) ]
                    , Table.td [] [ text thing.id ]
                    ]
            )
            model.things.list
        )


provisionModal : Model -> Html Msg
provisionModal model =
    Modal.config ClosePorvisionModal
        |> Modal.large
        |> Modal.hideOnBackdropClick True
        |> Modal.h4 [] [ text "Add thing" ]
        |> provisionModalBody model
        |> Modal.view model.provisionModalVisibility


provisionModalBody : Model -> (Modal.Config Msg -> Modal.Config Msg)
provisionModalBody model =
    Modal.body []
        [ Grid.container []
            [ Grid.row [] [ Grid.col [] [ provisionModalForm model ] ]
            , ModalMF.provisionModalButtons ProvisionThing ClosePorvisionModal
            ]
        ]


provisionModalForm : Model -> Html Msg
provisionModalForm model =
    ModalMF.modalForm
        [ ModalMF.FormRecord "name" SubmitName model.name model.name
        , ModalMF.FormRecord "metadata" SubmitMetadata model.metadata model.metadata
        ]



-- Edit modal


editModal : Model -> Html Msg
editModal model =
    Modal.config CloseEditModal
        |> Modal.large
        |> Modal.hideOnBackdropClick True
        |> Modal.h4 [] [ text (Helpers.parseString model.thing.name) ]
        |> editModalBody model
        |> Modal.view model.editModalVisibility


editModalBody : Model -> (Modal.Config Msg -> Modal.Config Msg)
editModalBody model =
    Modal.body []
        [ Grid.container []
            [ Grid.row []
                [ Grid.col []
                    [ editModalForm model
                    , ModalMF.modalDiv [ ( "id", model.thing.id ), ( "key", model.thing.key ) ]
                    ]
                ]
            , ModalMF.editModalButtons model.editMode UpdateThing EditThing (ShowEditModal model.thing) (RemoveThing model.thing.id) CloseEditModal
            ]
        ]


editModalForm : Model -> Html Msg
editModalForm model =
    if model.editMode then
        ModalMF.modalForm
            [ ModalMF.FormRecord "name" SubmitName (Helpers.parseString model.thing.name) model.name
            , ModalMF.FormRecord "metadata" SubmitMetadata (maybeJsonValueToString model.thing.metadata) model.metadata
            ]

    else
        ModalMF.modalDiv [ ( "name", Helpers.parseString model.thing.name ), ( "metadata", maybeJsonValueToString model.thing.metadata ) ]



-- JSON


thingDecoder : D.Decoder Thing
thingDecoder =
    D.map4 Thing
        (D.maybe (D.field "name" D.string))
        (D.field "id" D.string)
        (D.field "key" D.string)
        (D.maybe (D.field "metadata" jsonValueDecoder))


thingsDecoder : D.Decoder Things
thingsDecoder =
    D.map2 Things
        (D.field "things" (D.list thingDecoder))
        (D.field "total" D.int)


thingEncoder : Thing -> E.Value
thingEncoder thing =
    E.object
        [ ( "name", E.string (Helpers.parseString thing.name) )
        , ( "metadata", jsonValueEncoder (maybeJsonValueToJsonValue thing.metadata) )
        ]



-- HELPERS


resetEdit : Model -> Model
resetEdit model =
    { model | name = "", metadata = "" }


updateThingList : Model -> String -> ( Model, Cmd Msg )
updateThingList model token =
    ( model
    , Cmd.batch
        [ HttpMF.retrieve
            (B.relative [ paths.things ] (Helpers.buildQueryParamList model.offset model.limit))
            token
            RetrievedThings
            thingsDecoder
        , HttpMF.retrieve
            (B.relative [ paths.things, model.thing.id ] [])
            token
            RetrievedThing
            thingDecoder
        ]
    )
