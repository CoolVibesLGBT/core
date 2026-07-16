package constants

import (
	"github.com/google/uuid"
)

var zeroNamespace = uuid.Nil
var NameSpace = uuid.NewSHA1(zeroNamespace, []byte(APPLICATION_NAME))
