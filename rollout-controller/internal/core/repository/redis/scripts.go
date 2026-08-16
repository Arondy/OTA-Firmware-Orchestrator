package redis

import "github.com/redis/go-redis/v9"

var updateResultScript = redis.NewScript(`
local created = redis.call("SET", KEYS[1], "1", "NX", "EX", ARGV[1])
if created then
	return redis.call("INCR", KEYS[2])
end
return 0
`)
