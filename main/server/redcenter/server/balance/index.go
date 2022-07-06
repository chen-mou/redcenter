package balance

import "math/rand"

var Balance = map[string]func(...interface{}) int{

	"Hash": func(param ...interface{}) int {
		ip := param[0].(string)
		l := param[1].(int)
		res := 0
		for _, v := range ip {
			if v == '.' {
				continue
			}
			res += 10*res + int(v-'0')
		}
		return res % l
	},
	"Index": func(param ...interface{}) int {
		return param[0].(int)
	},
	"Weight": func(param ...interface{}) int {
		sum := 0
		for _, v := range param {
			sum += v.(int)
		}
		r := rand.Intn(sum)
		sum = 0
		for i, v := range param {
			sum += v.(int)
			if sum > r {
				return i
			}
		}
		return -1
	},
}
