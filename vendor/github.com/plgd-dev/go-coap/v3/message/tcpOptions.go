package message

// Signal CSM Option IDs
/*
   +-----+---+---+-------------------+--------+--------+---------+
   | No. | C | R | Name              | Format | Length | Default |
   +-----+---+---+-------------------+--------+--------+---------+
   |   2 |   |   | MaxMessageSize    | uint   | 0-4    | 1152    |
   |   4 |   |   | BlockWiseTransfer | empty  | 0      | (none)  |
   +-----+---+---+-------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	TCPMaxMessageSize    OptionID = 2
	TCPBlockWiseTransfer OptionID = 4
)

// Signal Ping/Pong Option IDs
/*
   +-----+---+---+-------------------+--------+--------+---------+
   | No. | C | R | Name              | Format | Length | Default |
   +-----+---+---+-------------------+--------+--------+---------+
   |   2 |   |   | Custody           | empty  | 0      | (none)  |
   +-----+---+---+-------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	TCPCustody OptionID = 2
)

// Signal Release Option IDs
/*
   +-----+---+---+---------------------+--------+--------+---------+
   | No. | C | R | Name                | Format | Length | Default |
   +-----+---+---+---------------------+--------+--------+---------+
   |   2 |   | x | Alternative-Address | string | 1-255  | (none)  |
   |   4 |   |   | Hold-Off            | uint3  | 0-3    | (none)  |
   +-----+---+---+---------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	TCPAlternativeAddress OptionID = 2
	TCPHoldOff            OptionID = 4
)

// Signal Abort Option IDs
/*
   +-----+---+---+---------------------+--------+--------+---------+
   | No. | C | R | Name                | Format | Length | Default |
   +-----+---+---+---------------------+--------+--------+---------+
   |   2 |   |   | Bad-CSM-Option      | uint   | 0-2    | (none)  |
   +-----+---+---+---------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/
const (
	TCPBadCSMOption OptionID = 2
)

var TCPSignalCSMOptionDefs = map[OptionID]OptionDef{
	TCPMaxMessageSize:    {ValueFormat: ValueUint, MinLen: 0, MaxLen: 4},
	TCPBlockWiseTransfer: {ValueFormat: ValueEmpty, MinLen: 0, MaxLen: 0},
}

var TCPSignalPingPongOptionDefs = map[OptionID]OptionDef{
	TCPCustody: {ValueFormat: ValueEmpty, MinLen: 0, MaxLen: 0},
}

var TCPSignalReleaseOptionDefs = map[OptionID]OptionDef{
	TCPAlternativeAddress: {ValueFormat: ValueString, MinLen: 1, MaxLen: 255},
	TCPHoldOff:            {ValueFormat: ValueUint, MinLen: 0, MaxLen: 3},
}

var TCPSignalAbortOptionDefs = map[OptionID]OptionDef{
	TCPBadCSMOption: {ValueFormat: ValueUint, MinLen: 0, MaxLen: 2},
}
