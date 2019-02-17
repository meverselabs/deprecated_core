package message_def

import "git.fleta.io/fleta/framework/message"

// types
var (
	BlockGenMessageType    = message.DefineType("fleta.BlockGen")
	BlockObSignMessageType = message.DefineType("fleta.BlockObSign")
	BlockReqMessageType    = message.DefineType("fleta.BlockReq")
	TransactionMessageType = message.DefineType("fleta.Transaction")
)
