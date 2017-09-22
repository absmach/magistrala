package manager

const Version = "1.0.0"

type Info struct {
	Version string
}

func getInfo() (Info, error) {
	info := Info{Version: Version}

	return info, nil
}
