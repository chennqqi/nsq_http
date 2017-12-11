# nsq_httpd

nsq consumer to httpd, like nsq_pubsub, only support stat/sub, but simple to use.


## usage


	Usage of ./nsq_httpd:
	  -channel string
	    	set nsq group
	  -http-address string
	    	<addr>:<port> to listen on for HTTP clients (default "127.0.0.1:8080")
	  -lookupd-http-address value
	    	lookupd HTTP address (may be given multiple times)
	  -max-in-flight int
	    	max number of messages to allow in flight (default 100)
	  -max-messages int
	    	return if got N messages in a single poll (default 1)
	  -nsqd-tcp-address value
	    	nsqd TCP address (may be given multiple times)
	  -timeout int
	    	return within N seconds if maxMessages not reached (default 10)
	  -topic string
	    	set nsq topic



	curl localhost:8080/sub?limit=xx&timeout=xx