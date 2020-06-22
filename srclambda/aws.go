package srclambda

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/go-errors/errors"
)

var ssmc *ssm.SSM

func init() {
	sess := session.Must(session.NewSession())
	ssmc = ssm.New(sess)
}

// SSMIOStrategy -
type SSMIOStrategy struct {
	KeyPrefix string
}

// Load -
func (a *SSMIOStrategy) Load(name string) (string, error) {
	t := true
	name = a.KeyPrefix + name
	param, err := ssmc.GetParameter(&ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: &t,
	})
	if err != nil {
		return "", err
	}
	return *param.Parameter.Value, nil
}

// Save -
func (a *SSMIOStrategy) Save(name string, value string) error {
	name = a.KeyPrefix + name
	t := true
	_, err := ssmc.PutParameter(&ssm.PutParameterInput{
		Name:      &name,
		Overwrite: &t,
		Value:     &value,
	})
	return err
}

// LoadBytes -
func (a *SSMIOStrategy) LoadBytes(name string) ([]byte, error) {
	s, e := a.Load(name)
	return []byte(s), e
}

// SaveBytes -
func (a *SSMIOStrategy) SaveBytes(name string, value []byte) error {
	return a.Save(name, string(value))
}

// Prompt -
func (a *SSMIOStrategy) Prompt() (string, error) {
	return "", errors.Errorf("Unable to get user input from within AWS Lambda")
}
