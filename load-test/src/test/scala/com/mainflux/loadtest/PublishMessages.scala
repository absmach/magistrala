package com.mainflux.loadtest

import io.circe._
import io.circe.parser._
import io.gatling.core.Predef._
import io.gatling.http.Predef._
import io.gatling.http.request.builder.HttpRequestBuilder.toActionBuilder
import scalaj.http.Http

import scala.concurrent.duration._

final class PublishMessages extends TestCase {
  override def prepareAndExecute(): SetUp = {
    def makeThing(token: String): (String, String) = {
      val thing = """{"type":"device", "name":"weio"}"""

      val id = Http(s"$ThingsURL/clients")
        .postData(thing)
        .header(HttpHeaderNames.Authorization, token)
        .header(HttpHeaderNames.ContentType, jsonType)
        .asString
        .headers(HttpHeaderNames.Location)(0).split("/")(2)

      val res = Http(s"$ThingsURL/things/$id")
        .header(HttpHeaderNames.Authorization, token)
        .header(HttpHeaderNames.ContentType, jsonType)
        .asString
        .body

      val key = parse(res).getOrElse(Json.Null).hcursor.downField("key").as[String].getOrElse("")

      (id, key)
    }

    def makeChannel(token: String): String = {
      val channel = """{"name":"mychan"}"""

      Http(s"$ThingsURL/channels")
        .postData(channel)
        .header(HttpHeaderNames.Authorization, token)
        .header(HttpHeaderNames.ContentType, jsonType)
        .asString
        .headers(HttpHeaderNames.Location)(0)
        .split("/")(2)
    }

    def connect(channel: String, thing: String, token: String): Unit = {
      Http(s"$ThingsURL/channels/$channel/things/$thing")
        .method("PUT")
        .header(HttpHeaderNames.Authorization, token)
        .asString
    }

    val message =
      """
        |[
        | {"bn":"some-base-name:","bt":1.276020076001e+09,"bu":"A","bver":5,"n":"voltage","u":"V","v":120.1},
        | {"n":"current","t":-5,"v":1.2},
        | {"n":"current","t":-4,"v":1.3}
        |]""".stripMargin

    val token = authenticate()
    val (thingID, thingKey) = makeThing(token)
    val channelID = makeChannel(token)

    connect(channelID, thingID, thingKey)

    val scn = scenario("publish message")
      .exec(http("publish message request")
        .post(s"/channels/$channelID/messages")
        .header(HttpHeaderNames.ContentType, "application/senml+json")
        .header(HttpHeaderNames.Authorization, thingKey)
        .body(StringBody(message))
        .check(status.is(202)))

    setUp(scn.inject(constantUsersPerSec(RequestsPerSecond) during 15.seconds)).protocols(httpProtocol(url()))
  }

  private def url(): String = System.getProperty("http", "http://localhost:8185")
}
