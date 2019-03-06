-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module Helpers exposing (buildQueryParamList, checkEntity, faIcons, fontAwesome, genPagination, isChecked, pageToOffset, parseString, response, validateInt, validateOffset)

import Bootstrap.Button as Button
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Grid.Row as Row
import Bootstrap.Utilities.Spacing as Spacing
import Html exposing (Html, div, hr, node, p, strong, text)
import Html.Attributes exposing (..)
import Http
import List.Extra
import Url.Builder as B



-- HTTP


response : String -> Html.Html msg
response resp =
    if String.length resp > 0 then
        Grid.row []
            [ Grid.col []
                [ hr [] []
                , p [] [ text ("response: " ++ resp) ]
                ]
            ]

    else
        Grid.row []
            [ Grid.col [] []
            ]



-- STRING


parseString : Maybe String -> String
parseString str =
    case str of
        Just s ->
            s

        Nothing ->
            ""



-- PAGINATION


buildQueryParamList : Int -> Int -> List B.QueryParameter
buildQueryParamList offset limit =
    [ B.int "offset" offset, B.int "limit" limit ]


validateInt : String -> Int -> Int
validateInt string default =
    case String.toInt string of
        Just num ->
            num

        Nothing ->
            default


pageToOffset : Int -> Int -> Int
pageToOffset page limit =
    (page - 1) * limit


validateOffset : Int -> Int -> Int -> Int
validateOffset offset total limit =
    if offset >= (total - 1) then
        (total - 1) - limit

    else
        offset


genPagination : Int -> (Int -> msg) -> Html msg
genPagination total msg =
    let
        pages =
            List.range 1 (Basics.ceiling (Basics.toFloat total / 10))

        cols =
            List.map
                (\page ->
                    Grid.col [] [ Button.button [ Button.roleLink, Button.attrs [ Spacing.ml1 ], Button.onClick (msg page) ] [ text (String.fromInt page) ] ]
                )
                pages
    in
    Grid.row [] cols



-- FONT-AWESOME


fontAwesome : Html msg
fontAwesome =
    node "link"
        [ rel "stylesheet"
        , href "https://use.fontawesome.com/releases/v5.7.2/css/all.css"
        , attribute "integrity" "sha384-fnmOCqbTlWIlj8LyTjo7mOUStjsKC4pOpQbqyi7RrhN7udi9RwhKkMHpvLbHG9Sr"
        , attribute "crossorigin" "anonymous"
        ]
        []


faIcons =
    { provision = class "fa fa-plus"
    , edit = class "fa fa-pen"
    , remove = class "fa fa-trash-alt"
    }



-- TABLE


checkEntity : String -> List String -> List String
checkEntity id checkedEntitiesIds =
    if List.member id checkedEntitiesIds then
        List.Extra.remove id checkedEntitiesIds

    else
        id :: checkedEntitiesIds


isChecked : String -> List String -> Bool
isChecked id checkedEntitiesIds =
    if List.member id checkedEntitiesIds then
        True

    else
        False
