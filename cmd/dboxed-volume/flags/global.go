package flags

type GlobalFlags struct {
	Debug bool `help:"Enable debugging mode"`

	ApiUrl   string  `help:"Specify the API url" default:"https://volumes.dboxed.io"`
	ApiToken *string `help:"Specify a static API token"`
}
