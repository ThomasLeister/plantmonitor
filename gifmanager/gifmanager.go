/*
 * GifManager:
 * Offers functions to retrieve GIF URLs from an online GIF platform, such as Giphy.
 */

package gifmanager

import (
	"log"
	"github.com/sanzaru/go-giphy"
)

type GiphyClient struct {
	Apiclient *libgiphy.Giphy
}

func (g *GiphyClient) Init(api_key string) {
	g.Apiclient = libgiphy.NewGiphy(api_key)
}

func (g *GiphyClient) GetGifURL(keywords string) (string, error) {
	dataRandom, err := g.Apiclient.GetRandom(keywords)
	if err != nil {
		log.Println("GifManager: ", err)
		return "", err
	}

	gifUrl := dataRandom.Data.Images.Original.Mp4
	log.Printf("GifManager: GIF URL: %+v\n", gifUrl)

	return gifUrl, nil
}
