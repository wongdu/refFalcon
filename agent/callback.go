package agent

var jcb JavaCallback

type JavaCallback interface {
	/*OneMethod()
	AnotherMethod(string)
	CapOneScreen()*/
	ChangeModeBits(string)
	AddOfflineRebootTimes()
}

func RegisterJavaCallback(cb JavaCallback) {
	jcb = cb
}

func cbChangeModeBits(filePath string) {
	if jcb != nil {
		jcb.ChangeModeBits(filePath)
	}
}

func cbAddOfflineRebootTimes() {
	if jcb != nil {
		jcb.AddOfflineRebootTimes()
	}
}
