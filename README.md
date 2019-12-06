# Chatbot Operator
A Kubernetes operator for implementing the platform of Chatbot as a Service. The users can use this platform to easily deploy multi-platform chatbot service without any programming.

# Prerequisites
* Make sure you already have a Kubernetes cluster (v1.16.0+).

## Setup Operator
To set up the chatbot operator, just deploy the YAML files from `deploy` folder.
```bash
# Clone the repository
$ git clone https://github.com/pohsienshih/chatbot-operator.git
$ cd chatbot-operator

# Setup the service account
$ kubectl apply -f deploy/service_account.yaml

# Setup the cluster rolebinding
$ kubectl apply -f deploy/clusterrole.yaml

# Deploy the operator
$ kubectl apply -f deploy/operator.yaml


# Deploy the CRD
$ kubectl apply -f deploy/crds/pohsienshih.com_bots_crd.yaml
$ kubectl apply -f deploy/crds/pohsienshih.com_messages_crd.yaml

```
## Deploy the Chatbot
In this operator, it will process the customized resource named `Bot` and deploy the specific chatbot. 

```bash
# Deploy the Line bot
$ kubectl apply -f example/line.yaml

# Deploy the Facebook Messenger bot
$ kubectl apply -f example/facebook.yaml

# Deploy the Telegram bot
$ kubectl apply -f example/telegram.yaml

```

The format of `Bot` resource is like following:
```yaml
# Line Bot
apiVersion: pohsienshih.com/v1
kind: Bot
metadata:
  name: example-linebot
  namespace: default
spec:
  bottype: line
  group: "group name"
  channelsecret: "Your channel secret"
  channeltoken: "Your channel token"
  size: 1
```

```yaml
# Facebook Messenger Bot
apiVersion: pohsienshih.com/v1
kind: Bot
metadata:
  name: example-messengerbot
  namespace: default
spec:
  bottype: facebook
  group: "group name"
  pagetoken: "Your page token"
  verifytoken: "Your verify token"
  size: 1
```
```yaml
# Telegram Bot
apiVersion: pohsienshih.com/v1
kind: Bot
metadata:
  name: example-telegrambot
  namespace: default
spec:
  bottype: telegram
  group: "group name"
  telegramtoken: "Your Telegram token"
  size: 1
```

Chatbot operator currently supports three platforms of the chatbot: Line, Facebook, and Telegram. You can easily deploy different types of bot by modifying the key `bottype`.

> Be noted that different chatbot platforms must configured different parameters.

`size` represents the number of the replicas of webhook deployment, you can enlarge the value to deploy more pods for load balance.

You can bind a bunch of chatbots together with the key `group`. If you leave this key blank, the value will same as the bot name by default.
This value will be used to appending the events for the chatbot. We will discuss the detail in the next section.

## Append the Events
The chatbot operator will also process the customized resource named `Message`. You can use this resource to easily add new events to your chatbot without any programming. 

```bash
# Append the Events
$ kubectl apply -f example/message.yaml
```

The format of `Message` resource is like following:
```yaml
apiVersion: pohsienshih.com/v1
kind: Message
metadata:
  name: example-message
  namespace: default
spec:
  botname:
  - name1
  - name2
  #...
  group:
  - group1
  - group2
  #...
  keyword: "Hello"
  response: "Welcome to Kubernetes :)."
```
The form of the event is key-value type. When the chatbot receives the matched keyword, it will reply to the string you configured.

The operator will use `botname` and `group` to get the matched chatbots. This makes you append events easily for one or more bot at the same time (The chatbot must at same namespace). If you don't want to use any of them, just leave it blank.

## Modify the Webhook Server
You can customize your webhook server by rebuilding the Docker image. But since we adopted etcd to the database in this operator,  make sure your webhook can work properly with etcd.

To get more information of the webhook images, please refer to the Dockerfile from the my dockerhub:

* [linebot-webhook-etcd](https://hub.docker.com/repository/docker/pohsienshih/linebot-webhook-etcd)
* [messengerbot-webhook-etcd](https://hub.docker.com/repository/docker/pohsienshih/messengerbot-webhook-etcd)
* [telegrambot-webhook-etcd](https://hub.docker.com/repository/docker/pohsienshih/telegrambot-webhook-etcd)
