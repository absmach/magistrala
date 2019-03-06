-- Copyright (c) 2019
-- Mainflux
--
-- SPDX-License-Identifier: Apache-2.0


module ModalMF exposing (FormRecord, editModalButtons, modalDiv, modalForm, provisionModalButtons)

import Bootstrap.Button as Button
import Bootstrap.Form as Form
import Bootstrap.Form.Input as Input
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Grid.Row as Row
import Bootstrap.Utilities.Spacing as Spacing
import Dict
import Helpers
import Html exposing (Html, div, hr, node, p, strong, text)
import Html.Attributes exposing (..)


button type_ msg txt =
    Button.button [ type_, Button.attrs [ Spacing.ml1 ], Button.onClick msg ] [ text txt ]


modalDiv paragraphList =
    div []
        (List.map
            (\paragraph ->
                p []
                    [ strong [] [ text (Tuple.first paragraph ++ ": ") ]
                    , text (Tuple.second paragraph)
                    ]
            )
            paragraphList
        )


type alias FormRecord msg =
    { text : String
    , msg : String -> msg
    , placeholder : String
    , value : String
    }


modalForm : List (FormRecord msg) -> Html msg
modalForm formList =
    Form.form []
        (List.map
            (\form ->
                Form.group []
                    [ Form.label [] [ strong [] [ text form.text ] ]
                    , Input.text [ Input.onInput form.msg, Input.attrs [ placeholder form.placeholder, value form.value ] ]
                    ]
            )
            formList
        )


editModalButtons mode updateMsg editMsg cancelMsg deleteMsg closeMsg =
    let
        lButton1 =
            if mode then
                button Button.outlinePrimary updateMsg "UPDATE"

            else
                button Button.outlinePrimary editMsg "EDIT"

        lButton2 =
            if mode then
                button Button.outlineDanger cancelMsg "CANCEL"

            else
                button Button.outlineDanger deleteMsg "DELETE"
    in
    Grid.row []
        [ Grid.col [ Col.attrs [ align "left" ] ]
            [ lButton1
            , lButton2
            ]
        , Grid.col [ Col.attrs [ align "right" ] ]
            [ button Button.outlineSecondary closeMsg "CLOSE"
            ]
        ]


provisionModalButtons provisionMsg closeMsg =
    Grid.row []
        [ Grid.col [ Col.attrs [ align "left" ] ]
            [ button Button.outlinePrimary provisionMsg "ADD"
            ]
        , Grid.col [ Col.attrs [ align "right" ] ]
            [ button Button.outlineSecondary closeMsg "CLOSE"
            ]
        ]
