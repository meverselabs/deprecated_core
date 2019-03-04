package message_def

import "github.com/fletaio/framework/message"

// types
var (
	BlockGenMessageType    = message.DefineType("fleta.BlockGen")
	BlockObSignMessageType = message.DefineType("fleta.BlockObSign")
	BlockReqMessageType    = message.DefineType("fleta.BlockReq")
	TransactionMessageType = message.DefineType("fleta.Transaction")
	PingMessageType        = message.DefineType("fleta.Ping")
)
