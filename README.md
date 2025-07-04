# gorabbit
RabbitMq queue consuming and producing 

# Consuming features
1. Manage subscribers on the fly using socket. 
2. Auto reconnect when connection failed.
3. Auto nack on panic and panic recover.
4. Multiple server and multiple queues implementation supports in config(yaml) files.
5. Callback registry. Allows you to create a callback for each queue.
6. Support for prefetch and streams

# Producing features
1. Reusing connection.
2. Implemented connection pool.
3. Manage max connections via a config file.
4. Free connection in idle status after the 10s

# Allowed commands
1. **consumer start all** - _start all consumer defined in registry_
2. **consumer start name_1 name_2** - _start specific consumers_
3. **consumer stop all** - _stop all consumer defined in registry_
4. **consumer stop** name_1 name_2 - _stop specific consumers_
5. **consumer restart all** - _restart all consumer defined in registry_
6. **consumer restart name_1 name_2** - _restart specific consumers_
7. **consumer status all** - _status of all consumer defined in registry_
8. **consumer status name_1 name_2** - _status of specific consumers_
9. **consumer set count N name_1 name_2** - _set count of subscribers for specific consumer_

# Example

```
echo "consumer status all" | nc localhost 3333
```

#### If you find this project useful or want to support the author, you can send tokens to any of these wallets
- Bitcoin: bc1qgx5c3n7q26qv0tngculjz0g78u6mzavy2vg3tf
- Ethereum: 0x62812cb089E0df31347ca32A1610019537bbFe0D
- Dogecoin: DET7fbNzZftp4sGRrBehfVRoi97RiPKajV