package registry

import "cnrancher.io/image-tools/utils"

// RunCommandFunc specifies the custom function to run command for registry.
//
// Only used for testing purpose!
var RunCommandFunc utils.RunCmdFuncType = nil
