package irc

import "regexp"

const (
	// nStringsAssumed is the number of channels assumed to be in each irc message
	// if this number is too small, there could be memory thrashing due to append
	nChannelsAssumed = 1
)

// ChannelFinder stores a cached regexp generated by CreateChannelFinder in
// order to scan a string for potential channel entries.
type ChannelFinder struct {
	channelRegexp *regexp.Regexp
}

// CreateChannelFinder safely builds a regex from the Chantypes that are passed
// in. For flexibility it can be initialized with a string, but ideally
// ProtoCaps.Chantypes parsed from the server's message will be used.
func CreateChannelFinder(types string) (*ChannelFinder, error) {
	c := &ChannelFinder{}
	safetypes := ""
	for _, c := range types {
		safetypes += string(`\`) + string(c)
	}
	regex, err := regexp.Compile(`[` + safetypes + `][^\s,]+`)
	if err == nil {
		c.channelRegexp = regex
		return c, nil
	}
	return nil, err
}

// FindChannels retrieves all the channels in the string using a cached regex.
// Calls to this will fail if the ChannelFinder was not initialized with
// CreateChannelFinder
func (c *ChannelFinder) FindChannels(msg string) []string {
	channels := make([]string, 0, nChannelsAssumed)

	for _, v := range c.channelRegexp.FindAllString(msg, -1) {
		channels = append(channels, v)
	}

	return channels
}
