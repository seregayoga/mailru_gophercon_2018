package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// глобальные переменные запрещены
// cgo запрещен

type User struct {
	ID       int
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Hits     []string `json:"hits"`
	Name     string   `json:"name"`
}

type Users []*User

func (u Users) Len() int {
	return len(u)
}
func (u Users) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
func (u Users) Less(i, j int) bool {
	return u[i].ID < u[j].ID
}

func Fast(in io.Reader, out io.Writer, networks []string) {
	r := bufio.NewReader(in)

	re, _ := regexp.Compile(`Chrome/(60.0.3112.90|52.0.2743.116|57.0.2987.133)`)

	nets := make([]*net.IPNet, len(networks))
	for i, n := range networks {
		_, nets[i], _ = net.ParseCIDR(n)
	}

	usersCh := make(chan *User, 4)
	ID := 0

	wg := &sync.WaitGroup{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}

		ID++

		wg.Add(1)
		go func(line string, ID int) {
			defer wg.Done()

			user := &User{}
			json.Unmarshal([]byte(line), user)
			user.ID = ID

			netCount := 0
			for _, hit := range user.Hits {
				ip := net.ParseIP(hit)
				for _, net := range nets {
					if net.Contains(ip) {
						netCount++

						if netCount == 3 {
							break
						}
					}
				}

				if netCount == 3 {
					break
				}
			}

			if netCount < 3 {
				return
			}

			emailCount := 0
			for _, ua := range user.Browsers {
				if re.MatchString(ua) {
					emailCount++

					if emailCount == 3 {
						break
					}
				}
			}

			if emailCount < 3 {
				return
			}

			user.Email = strings.Replace(user.Email, "@", " [at] ", 1)

			usersCh <- user
		}(line, ID)
	}

	go func() {
		wg.Wait()
		close(usersCh)
	}()

	users := Users{}
	for user := range usersCh {
		users = append(users, user)
	}

	sort.Sort(users)
	fmt.Fprintf(out, "Total: %d\n", len(users))
	for _, user := range users {
		fmt.Fprintf(out, fmt.Sprintf("[%d] %s <%s>\n", user.ID, user.Name, user.Email))
	}
}
