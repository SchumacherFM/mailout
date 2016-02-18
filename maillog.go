package mailout

type mailogger interface {
	Log([]byte)
}

func newMailLogger(path string) mailogger {
	return nil
}
