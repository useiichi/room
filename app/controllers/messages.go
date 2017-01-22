package controllers

import "github.com/revel/revel"

type Messages struct {
	*revel.Controller
}

func (c Messages) New() revel.Result {
	return c.Render()
}

func (c Messages) MessageCreated(myName string) revel.Result {
	c.Validation.Required(myName).Message("Your name is required!")
	c.Validation.MinSize(myName, 3).Message("Your name is not long enough!")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Messages.New)
	}

	return c.Render(myName)
}
