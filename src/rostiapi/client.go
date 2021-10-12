package rostiapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const apiHost = "admin.rosti.cz"

// Client groups all Rosti API calls
type Client struct {
	Timeout    int       // Default is 10 seconds
	Token      string    // token provided by Fakturoid
	CompanyID  uint      // Company ID
	ExtraError io.Writer // where to send extra error information
}

func (c *Client) getTimeout() time.Duration {
	timeout := 60
	if c.Timeout != 0 {
		timeout = c.Timeout
	}

	return time.Duration(timeout) * time.Second
}

func (c *Client) call(method string, path string, payload []byte) ([]byte, int, error) {
	client := &http.Client{
		Timeout: c.getTimeout(),
	}

	payloadReader := bytes.NewReader(payload)

	req, err := http.NewRequest(method, "https://"+apiHost+"/api/v1/"+path, payloadReader)
	if err != nil {
		return []byte{}, 0, err
	}

	req.Header.Add("authorization", "Token "+c.Token)
	req.Header.Add("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// GetApps returns list of all applications belonging to the
func (c *Client) GetApps() ([]App, error) {
	apps := []App{}

	body, statusCode, err := c.call("GET", strconv.Itoa(int(c.CompanyID))+"/"+"apps/", []byte(""))
	if err != nil {
		return apps, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return apps, errors.New("non-200 HTTP status code returned")
	}

	err = json.Unmarshal(body, &apps)
	if err != nil {
		return apps, errors.New("cannot parse response from the server")
	}

	return apps, nil
}

// GetApp returns one application based on its ID
func (c *Client) GetApp(id uint) (App, error) {
	app := App{}

	body, statusCode, err := c.call("GET", strconv.Itoa(int(c.CompanyID))+"/"+"apps/"+strconv.Itoa(int(id))+"/", []byte(""))
	if err != nil {
		return app, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return app, errors.New("non-200 HTTP status code returned")
	}

	err = json.Unmarshal(body, &app)
	if err != nil {
		return app, errors.New("cannot parse response from the server")
	}

	return app, nil
}

// CreateApp creates an application
func (c *Client) CreateApp(app *App) (*App, error) {
	body, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("problem while encoding App into json: %w", err)
	}

	body, statusCode, err := c.call("POST", strconv.Itoa(int(c.CompanyID))+"/"+"apps/", body)
	if statusCode != 200 {
		responseError := ErrorResponse{}
		err = json.Unmarshal(body, &responseError)
		if err != nil {
			if c.ExtraError != nil {
				c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
			}
			return nil, fmt.Errorf("problem while decoding error message: %w", err)
		}

		log.Println("Returned error:", responseError.Errors)
		return nil, errors.New(strconv.Itoa(statusCode) + " HTTP status code returned (" + responseError.Message + ")")
	}

	createdApp := App{}

	err = json.Unmarshal(body, &createdApp)
	if err != nil {
		return &createdApp, fmt.Errorf("problem while decoding app structure: %w", err)
	}

	return &createdApp, nil
}

// UpdateApp updates parameters of an application
func (c *Client) UpdateApp(app *App) (*App, error) {
	body, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("problem while encoding App into json: %w", err)
	}

	body, statusCode, err := c.call("PUT", strconv.Itoa(int(c.CompanyID))+"/"+"apps/"+strconv.Itoa(int(app.ID))+"/", body)
	if statusCode != 200 {
		responseError := ErrorResponse{}
		err = json.Unmarshal(body, &responseError)
		if err != nil {
			if c.ExtraError != nil {
				c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
			}
			return nil, fmt.Errorf("problem while decoding error message: %w", err)
		}

		log.Println("Returned error:", responseError.Errors)
		return nil, errors.New(strconv.Itoa(statusCode) + " HTTP status code returned (" + responseError.Message + ")")
	}

	updatedApp := App{}

	err = json.Unmarshal(body, &updatedApp)
	if err != nil {
		return &updatedApp, fmt.Errorf("problem while decoding app structure: %w", err)
	}

	return &updatedApp, nil
}

// DeleteApp deletes application the system
func (c *Client) DeleteApp(id uint) error {
	body, statusCode, err := c.call("DELETE", strconv.Itoa(int(c.CompanyID))+"/"+"apps/"+strconv.Itoa(int(id))+"/", []byte(""))
	if err != nil {
		return err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return errors.New("non-200 HTTP status code returned")
	}

	return nil
}

// DoApp calls action on the application identified by ID parameter. Action parameter can be start, stop, restart or rebuild.
func (c *Client) DoApp(id uint, action string) error {
	body, err := json.Marshal(Action{Action: action})
	if err != nil {
		return fmt.Errorf("problem while encoding action structure into JSON: %w", err)
	}

	body, statusCode, err := c.call("PUT", strconv.Itoa(int(c.CompanyID))+"/"+"apps-action/"+strconv.Itoa(int(id))+"/", body)
	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}

		responseError := ErrorResponse{}
		err = json.Unmarshal(body, &responseError)
		if err != nil {
			return fmt.Errorf("problem while decoding error message: %w", err)
		}

		return errors.New(strconv.Itoa(statusCode) + " HTTP status code returned (" + responseError.Message + ")")
	}

	return nil
}

// GetPlans returns list of plans available in the admin
func (c *Client) GetPlans() ([]Plan, error) {
	plans := []Plan{}

	body, statusCode, err := c.call("GET", strconv.Itoa(int(c.CompanyID))+"/"+"plans/", []byte(""))
	if err != nil {
		return plans, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return plans, errors.New("non-200 HTTP status code returned")
	}

	err = json.Unmarshal(body, &plans)
	if err != nil {
		return plans, errors.New("cannot parse response from the server")
	}

	return plans, nil
}

// GetCompanies returns list of companies where the token has access to
func (c *Client) GetCompanies() ([]Company, error) {
	companies := []Company{}

	body, statusCode, err := c.call("GET", "companies/", []byte(""))
	if err != nil {
		return companies, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return companies, errors.New("non-200 HTTP status code returned (" + string(body) + ")")
	}

	err = json.Unmarshal(body, &companies)
	if err != nil {
		return companies, errors.New("cannot parse response from the server")
	}

	return companies, nil
}

// GetRuntimes returns list of runtimes available in the admin
func (c *Client) GetRuntimes() ([]Runtime, error) {
	runtimes := []Runtime{}

	body, statusCode, err := c.call("GET", strconv.Itoa(int(c.CompanyID))+"/"+"runtimes/", []byte(""))
	if err != nil {
		return runtimes, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return runtimes, errors.New("non-200 HTTP status code returned")
	}

	err = json.Unmarshal(body, &runtimes)
	if err != nil {
		return runtimes, errors.New("cannot parse response from the server")
	}

	return runtimes, nil
}

// GetAppStatus returns information about running application
func (c *Client) GetAppStatus(id uint) (AppStatus, error) {
	appStatus := AppStatus{}

	body, statusCode, err := c.call("GET", strconv.Itoa(int(c.CompanyID))+"/"+"apps-status/"+strconv.Itoa(int(id))+"/", []byte(""))
	if err != nil {
		return appStatus, err
	}

	if statusCode != 200 {
		if c.ExtraError != nil {
			c.ExtraError.Write([]byte(fmt.Sprintf("Response body: %b", body)))
		}
		return appStatus, errors.New("non-200 HTTP status code returned")
	}

	err = json.Unmarshal(body, &appStatus)
	if err != nil {
		return appStatus, errors.New("cannot parse response from the server")
	}

	return appStatus, nil
}
