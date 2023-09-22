package main

import (
	"fmt"
	"bytes"
	"os/exec"
	"strings"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/ssh"
)

var lstRules []string

func main() {
	getIptablesRules()
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		rulesList := "<h2>Raw Rules:</h2><ul>"
		for i, rawRule := range lstRules {
			rulesList += fmt.Sprintf("<li>%s <a href='/deleteRawRule/%d'>Delete</a></li>", rawRule, i)
		}
		rulesList += "</ul>"

		c.Type("html")
		return c.SendString(rulesList)
	})

	app.Get("/deleteRawRule/:index", func(c *fiber.Ctx) error {
		index := c.Params("index")
		var idx int
		fmt.Sscanf(index, "%d", &idx)
		if idx >= 0 && idx < len(lstRules) {
			deleteIptablesRule(idx)
			getIptablesRules()
		}
		return c.Redirect("/")
	})

	app.Get("/traffic", func(c *fiber.Ctx) error {
		cmd := exec.Command("sudo", "iptables", "-nvL")
		output, err := cmd.Output()
		if err != nil {
			return c.SendString("Error fetching iptables data: " + err.Error())
		}
		c.Type("html")
		return c.SendString("<pre>" + string(output) + "</pre>")
	})

	app.Get("/remotetable", func(c *fiber.Ctx) error {
		formHTML := `
		<form action="/fetchIptables" method="post">
			Username: <input type="text" name="username"><br>
			Password: <input type="password" name="password"><br>
			Server IP: <input type="text" name="server_ip"><br>
			<input type="submit" value="Fetch iptables">
		</form>
		`
		c.Type("html")
		return c.SendString(formHTML)
	})

	app.Post("/fetchIptables", func(c *fiber.Ctx) error {
		username := c.FormValue("username")
		password := c.FormValue("password")
		serverIP := c.FormValue("server_ip")

		output, err := fetchRemoteIptables(username, password, serverIP)
		if err != nil {
			return c.Status(500).SendString("Error fetching iptables: " + err.Error())
		}

		return c.SendString("<pre>" + output + "</pre>")
	})


	app.Listen(":60123")
}

func getIptablesRules() {
	lstRules = nil // Clear the rules list
	cmd := exec.Command("iptables", "-L", "--line-numbers", "-n", "-v")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error fetching iptables rules:", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		lstRules = append(lstRules, line)
	}
}

func deleteIptablesRule(lineNumber int) {
	// Assuming the rules are in the INPUT chain. Adjust if needed.
	fmt.Println("Attempting to delete rule at line number:", lineNumber)

	cmd := exec.Command("iptables", "-D", "INPUT", fmt.Sprintf("%d", lineNumber))
	output, err := cmd.CombinedOutput() // This captures both standard output and error output
	if err != nil {
		fmt.Println("Error deleting iptables rule:", err)
		fmt.Println("Command output:", string(output)) // Print the detailed error message
	}
}




func fetchRemoteIptables(username, password, serverIP string) (string, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", serverIP+":22", config)
	if err != nil {
		return "", err
	}
	defer client.Close()

	var b bytes.Buffer
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	session.Stdout = &b
	if err := session.Run("sudo iptables -nvL"); err != nil {
		return "", err
	}
	return b.String(), nil
}

