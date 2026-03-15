package contract

import shared "github.com/mopeyjellyfish/hookr/internal/contractspec"

var (
	ErrContractNameEmpty    = shared.ErrContractNameEmpty
	ErrContractHashEmpty    = shared.ErrContractHashEmpty
	ErrMethodIDDuplicate    = shared.ErrMethodIDDuplicate
	ErrMethodNameDuplicate  = shared.ErrMethodNameDuplicate
	ErrMethodNameEmpty      = shared.ErrMethodNameEmpty
	ErrMethodRequestMissing = shared.ErrMethodRequestMissing
	ErrMethodReplyMissing   = shared.ErrMethodReplyMissing
)

type (
	MethodID = shared.MethodID
	Method   = shared.Method
	Schema   = shared.Schema
)
