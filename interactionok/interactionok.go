package interactionok

var RegisteredPlugins []string

func RegisterPlugin(s string) {
	RegisteredPlugins = append(RegisteredPlugins, s)
}
