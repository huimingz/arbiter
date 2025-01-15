package lua

// TryLock is the Lua script for trying to acquire a lock
const TryLock = `
if redis.call('exists', KEYS[1]) == 0 then
    redis.call('hset', KEYS[1], 'owner', ARGV[1])
    redis.call('pexpire', KEYS[1], ARGV[2])
    return 1
end
return 0
`

// Unlock is the Lua script for releasing a lock
const Unlock = `
if redis.call('hget', KEYS[1], 'owner') == ARGV[1] then
    return redis.call('del', KEYS[1])
else
    return 0
end
`

// Refresh is the Lua script for refreshing a lock's expiration
const Refresh = `
if redis.call('hget', KEYS[1], 'owner') == ARGV[1] then
    return redis.call('pexpire', KEYS[1], ARGV[2])
end
return 0
`
