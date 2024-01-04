package systray

import (
	"bytes"

	"fyne.io/systray"
)

type cachedMenuItem struct {
	menuItem *systray.MenuItem

	title   string
	tooltip string

	checked  bool
	disabled bool
	hidden   bool

	icon         []byte
	templateIcon []byte

	ClickedCh <-chan struct{}
}

func AddMenuItem(title string, tooltip string) *cachedMenuItem {
	menuItem := systray.AddMenuItem(title, tooltip)
	return &cachedMenuItem{
		menuItem:  menuItem,
		title:     title,
		tooltip:   tooltip,
		ClickedCh: menuItem.ClickedCh,
	}
}

func AddMenuItemCheckbox(title string, tooltip string, checked bool) *cachedMenuItem {
	menuItem := systray.AddMenuItemCheckbox(title, tooltip, checked)
	return &cachedMenuItem{
		menuItem:  menuItem,
		title:     title,
		tooltip:   tooltip,
		checked:   checked,
		ClickedCh: menuItem.ClickedCh,
	}
}

func (c *cachedMenuItem) AddSeparator() {
	c.menuItem.AddSeparator()
}

func (c *cachedMenuItem) AddSubMenuItem(title string, tooltip string) *cachedMenuItem {
	menuItem := c.menuItem.AddSubMenuItem(title, tooltip)
	return &cachedMenuItem{
		menuItem:  menuItem,
		title:     title,
		tooltip:   tooltip,
		ClickedCh: menuItem.ClickedCh,
	}
}

func (c *cachedMenuItem) AddSubMenuItemCheckbox(title string, tooltip string, checked bool) *cachedMenuItem {
	menuItem := c.menuItem.AddSubMenuItemCheckbox(title, tooltip, checked)
	return &cachedMenuItem{
		menuItem:  menuItem,
		title:     title,
		tooltip:   tooltip,
		checked:   checked,
		ClickedCh: menuItem.ClickedCh,
	}
}

func (c *cachedMenuItem) Check() {
	if !c.checked {
		c.checked = true
		c.menuItem.Check()
	}
}

func (c *cachedMenuItem) Checked() bool {
	return c.checked
}

func (c *cachedMenuItem) Disable() {
	if !c.disabled {
		c.disabled = true
		c.menuItem.Disable()
	}
}

func (c *cachedMenuItem) Disabled() bool {
	return c.disabled
}

func (c *cachedMenuItem) Enable() {
	if c.disabled {
		c.disabled = false
		c.menuItem.Enable()
	}
}

func (c *cachedMenuItem) Hide() {
	if !c.hidden {
		c.hidden = true
		c.menuItem.Hide()
	}
}

func (c *cachedMenuItem) Remove() {
	c.menuItem.Remove()
}

func (c *cachedMenuItem) SetIcon(iconBytes []byte) {
	if !bytes.Equal(c.icon, iconBytes) {
		c.icon = iconBytes
		c.menuItem.SetIcon(iconBytes)
	}
}

func (c *cachedMenuItem) SetTemplateIcon(templateIconBytes []byte, regularIconBytes []byte) {
	if !bytes.Equal(c.icon, regularIconBytes) || !bytes.Equal(c.templateIcon, templateIconBytes) {
		c.icon = regularIconBytes
		c.templateIcon = templateIconBytes
		c.menuItem.SetTemplateIcon(templateIconBytes, regularIconBytes)
	}
}

func (c *cachedMenuItem) SetTitle(title string) {
	if c.title != title {
		c.title = title
		c.menuItem.SetTitle(title)
	}
}

func (c *cachedMenuItem) SetTooltip(tooltip string) {
	if c.tooltip != tooltip {
		c.tooltip = tooltip
		c.menuItem.SetTooltip(tooltip)
	}
}

func (c *cachedMenuItem) Show() {
	if c.hidden {
		c.hidden = false
		c.menuItem.Show()
	}
}

func (c *cachedMenuItem) String() string {
	return c.menuItem.String()
}

func (c *cachedMenuItem) Uncheck() {
	if c.checked {
		c.checked = false
		c.menuItem.Uncheck()
	}
}
