package mailconsumer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
)

import (
	"github.com/mitchellh/mapstructure"
	"github.com/openware/postmaster/pkg/consumer"
	"github.com/openware/postmaster/pkg/eventapi"
	"github.com/openware/postmaster/pkg/utils"
)

const emailConfirmationRoutingKey = "user.email.confirmation.token"

func HandleEmailConfirmationEvent(r eventapi.Event) error {
	acc := EmailConfirmationEvent{}

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "json",
		Result:           &acc,
		WeaklyTypedInput: true,
	})

	if err != nil {
		return err
	}

	if err := dec.Decode(r); err != nil {
		return err
	}

	tpl, err := template.ParseFiles("templates/sign_up.tpl")
	if err != nil {
		return err
	}

	buff := bytes.Buffer{}
	if err := tpl.Execute(&buff, acc); err != nil {
		return err
	}

	apiKey := utils.MustGetEnv("SENDGRID_API_KEY")

	email := eventapi.Email{
		FromAddress: utils.GetEnv("SENDER_EMAIL", "noreply@postmaster.com"),
		FromName:    utils.GetEnv("SENDER_NAME", "postmaster"),
		Subject:     "Email Confirmation Instructions",
		Reader:      bytes.NewReader(buff.Bytes()),
	}

	if _, err := email.Send(apiKey, acc.User.Email); err != nil {
		return err
	}

	return nil
}

func EmailConfirmationHandler(amqpURI string) {
	c := consumer.New(amqpURI, Exchange, emailConfirmationRoutingKey)
	queueName := fmt.Sprintf("postmaster.%s.consumer", emailConfirmationRoutingKey)
	queue := c.DeclareQueue(queueName)
	c.BindQueue(queue)

	deliveries, err := c.Channel.Consume(
		queue.Name,
		c.Tag,
		true,
		true,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Panicf("Consuming: %s", err.Error())
	}

	go func() {
		for delivery := range deliveries {
			jwtReader, err := eventapi.DeliveryAsJWT(delivery)

			if err != nil {
				log.Println(err)
				return
			}

			jwt, err := ioutil.ReadAll(jwtReader)
			if err != nil {
				log.Println(err)
				return
			}

			log.Printf("Token: %s\n", string(jwt))

			claims, err := eventapi.ParseJWT(string(jwt), eventapi.ValidateJWT)
			if err != nil {
				log.Println(err)
				return
			}

			if err := HandleEmailConfirmationEvent(claims.Event); err != nil {
				log.Printf("Consuming: %s\n", err.Error())
			}
		}
	}()
}
