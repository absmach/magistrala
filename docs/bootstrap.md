## Bootstrap

`Bootstrapping` refers to a self-starting process that is supposed to proceed without external input.
Mainflux platform supports bootstrapping process, but some of the preconditions need to be fulfilled in advance. The device can trigger a bootstrap when:
- device contains only bootstrap credentials and no Mainflux credentials
- device, for any reason, fails to start a communication with the configured Mainflux services (server not responding, authentication failure, etc..).
- device, for any reason, wants to update its configuration

> Bootstrapping and provisioning are two different procedures. Provisioning refers to entities management while bootstrapping is related to entity configuration.


<style>
    .carousel {
        margin-left: 15%;
        margin-right: 15%;
    }

    ul.slides {
        display: block;
        position: relative;
        height: 600px;
        margin: 0;
        padding: 0;
        overflow: hidden;
        list-style: none;
    }

    .slides * {
        user-select: none;
        -ms-user-select: none;
        -moz-user-select: none;
        -khtml-user-select: none;
        -webkit-user-select: none;
        -webkit-touch-callout: none;
    }

    ul.slides input {
        display: none;
    }

    .slide-container {
        display: block;
    }

    .slide-image {
        display: block;
        position: absolute;
        width: 100%;
        height: 100%;
        top: 0;
        opacity: 0;
        transition: all .7s ease-in-out;
    }

    .slide-image img {
        width: auto;
        min-width: 100%;
        height: 100%;
    }

    .carousel-controls {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        z-index: 999;
        font-size: 100px;
        line-height: 600px;
        color: #d2dade;
    }

    .carousel-controls label {
        display: none;
        position: absolute;
        padding: 0 20px;
        opacity: 0;
        transition: opacity .2s;
        cursor: pointer;
    }

    .slide-image:hover+.carousel-controls label {
        opacity: 0.5;
    }

    .carousel-controls label:hover {
        opacity: 1;
    }

    .carousel-controls .prev-slide {
        width: 49%;
        text-align: left;
        left: 0;
    }

    .carousel-controls .next-slide {
        width: 49%;
        text-align: right;
        right: 0;
    }

    .carousel-dots {
        position: absolute;
        left: 0;
        right: 0;
        bottom: 20px;
        z-index: 999;
        text-align: center;
    }

    .carousel-dots .carousel-dot {
        display: inline-block;
        width: 30px;
        height: 30px;
        border-radius: 50%;
        background-color: #ecf0f2;
        opacity: 0.5;
        margin: 10px;
    }

    input:checked+.slide-container .slide-image {
        opacity: 1;
        transform: scale(1);
        transition: opacity 1s ease-in-out;
    }

    input:checked+.slide-container .carousel-controls label {
        display: block;
    }

    input#img-1:checked~.carousel-dots label#img-dot-1,
    input#img-2:checked~.carousel-dots label#img-dot-2,
    input#img-3:checked~.carousel-dots label#img-dot-3,
    input#img-4:checked~.carousel-dots label#img-dot-4,
    input#img-5:checked~.carousel-dots label#img-dot-5,
    input#img-6:checked~.carousel-dots label#img-dot-6 {
        opacity: 1;
    }

    input:checked+.slide-container .nav label {
        display: block;
    }
</style>
<div>
    <div class="carousel">
        <ul class="slides">
            <input type="radio" name="radio-buttons" id="img-1" checked />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/1.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-3" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-2" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <input type="radio" name="radio-buttons" id="img-2" />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/2.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-1" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-3" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <input type="radio" name="radio-buttons" id="img-3" />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/3.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-2" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-4" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <input type="radio" name="radio-buttons" id="img-4" />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/4.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-3" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-5" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <input type="radio" name="radio-buttons" id="img-5" />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/5.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-4" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-6" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <input type="radio" name="radio-buttons" id="img-6" />
            <li class="slide-container">
                <div class="slide-image">
                    <img src="img/bootstrap/6.png">
                </div>
                <div class="carousel-controls">
                    <label for="img-5" class="prev-slide">
                        <span>&lsaquo;</span>
                    </label>
                    <label for="img-1" class="next-slide">
                        <span>&rsaquo;</span>
                    </label>
                </div>
            </li>
            <div class="carousel-dots">
                <label for="img-1" class="carousel-dot" id="img-dot-1"></label>
                <label for="img-2" class="carousel-dot" id="img-dot-2"></label>
                <label for="img-3" class="carousel-dot" id="img-dot-3"></label>
                <label for="img-4" class="carousel-dot" id="img-dot-4"></label>
                <label for="img-5" class="carousel-dot" id="img-dot-5"></label>
                <label for="img-6" class="carousel-dot" id="img-dot-6"></label>
            </div>
        </ul>
    </div>
</div>


### Configuration

The configuration of Mainflux thing consists of two major parts:

- The list of Mainflux channels the thing is connected to
- Custom configuration related to the specific thing

Also, the configuration contains an external ID and external key, which will be explained later.
In order to enable the thing to start bootstrapping process, the user needs to upload a valid configuration for that specific thing. This can be done using the following HTTP request:

```
curl -s -S -i -X POST -H "Authorization: <user_token>" -H "Content-Type: application/json" http://localhost:8200/things/configs -d '{
        "external_id":"09:6:0:sb:sa",
        "thing_id": "1b9b8fae-9035-4969-a240-7fe5bdc0ed28",
        "external_key":"key",
        "name":"some",
        "channels":[
                "c3642289-501d-4974-82f2-ecccc71b2d83",
                "cd4ce940-9173-43e3-86f7-f788e055eb14",
                "ff13ca9c-7322-4c28-a25c-4fe5c7b753fc",
                "c3642289-501d-4974-82f2-ecccc71b2d82"
],
        "content": "config..."
}'
```

In this example, `channels` field represents the list of Mainflux channel IDs the thing is connected to. These channels need to be provisioned before the configuration is uploaded. Field `content` represents custom configuration. This custom configuration contains parameters that can be used to set up the thing. It can also be empty if no additional set up is needed. Field `name` is human readable name and `thing_id` is an ID of the Mainflux thing. This field is not required. If `thing_id` is empty, corresponding Mainflux thing will be created implicitly and its ID will be sent as a part of `Location` header of the response.

There are two more fields: `external_id` and `external_key`. External ID represents an ID of the device that corresponds to the given thing. For example, this can be a MAC address or the serial number of the device. The external key represents the device key. This is the secret key that's safely stored on the device and it is used to authorize the thing during the bootstrapping process. Please note that external ID and external key and Mainflux ID and Mainflux key are _completely different concepts_. External id and key are only used to authenticate a device that corresponds to the specific Mainflux thing during the bootstrapping procedure.

### Bootstrapping

Currently, the bootstrapping procedure is executed over the HTTP protocol. Bootstrapping is nothing else but fetching and applying the configuration that corresponds to the given Mainflux thing. In order to fetch the configuration, _the thing_ needs to send a bootstrapping request:

```
curl -s -S -i -H "Authorization: <external_key>" http://localhost:8200/things/bootstrap/<external_id>
```

The response body should look something like:

```
{
   "mainflux_id":"7c9df5eb-d06b-4402-8c1a-df476e4394c8",
   "mainflux_key":"86a4f870-eba4-46a0-bef9-d94db2b64392",
   "mainflux_channels":[
      {
         "id":"ff13ca9c-7322-4c28-a25c-4fe5c7b753fc",
         "name":"some channel",
         "metadata":{
            "operation":"someop",
            "type":"metadata"
         }
      },
      {
         "id":"925461e6-edfb-4755-9242-8a57199b90a5",
         "name":"channel1",
         "metadata":{
            "type":"control"
         }
      }
   ],
   "content":"config..."
}
```

The response consists of an ID and key of the Mainflux thing, the list of channels and custom configuration (`content` field). The list of channels contains not just channel IDs, but the additional Mainflux channel data (`name` and `metadata` fields), as well.

### Enabling and disabling things

Uploading configuration does not automatically connect thing to the given list of channels. In order to connect the thing to the channels, user needs to send the following HTTP request:

```
curl -s -S -i -X PUT -H "Authorization: <user_token>" -H "Content-Type: application/json" http://localhost:8200/things/state/<thing_id> -d '{"state": 1}'
```

In order to disconnect, the same request should be sent with the value of `state` set to 0.

For more information about Bootstrap API, please check out the [API documentation](https://github.com/mainflux/mainflux/blob/master/bootstrap/swagger.yml).
