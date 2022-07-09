/*
 * Helper functions for messenger package
 */

package messenger

import (
	"fmt"
	"net/mail"
	"strings"
)

/*
 * Convert xmppMessage.From to sender because "From" is not necessarily in JID form but one of:
 *	    <user>@<server>.tld/Resource
 *	    <server>.tld
 *	    <user>@<server>.tld
 */
func senderFromToJID(senderFrom string) (string, error) {
	var senderResource *mail.Address
	var senderJID string

	senderResource, err := mail.ParseAddress(senderFrom)
	if err != nil {
		return "", fmt.Errorf("'from' string '%s' cannot be parsed as JID", senderFrom)
	}

	// ParseAddress will also return the <bla>/Resource part, so remove the resource part by splitting at "/"
	senderJID = strings.Split(senderResource.Address, "/")[0]

	return senderJID, nil
}
