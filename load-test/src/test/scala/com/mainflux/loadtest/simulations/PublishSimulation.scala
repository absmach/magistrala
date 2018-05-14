package com.mainflux.loadtest.simulations

import scala.concurrent.duration._
import scalaj.http.Http
import io.gatling.core.Predef._
import io.gatling.http.Predef._
import io.gatling.jdbc.Predef._
import io.circe._
import io.circe.generic.auto._
import io.circe.parser._
import io.circe.syntax._
import PublishSimulation._
import io.gatling.http.protocol.HttpProtocolBuilder.toHttpProtocol
import io.gatling.http.request.builder.HttpRequestBuilder.toActionBuilder
import com.mainflux.loadtest.simulations.Constants._

class PublishSimulation extends Simulation {

  // Register user
  Http(s"${UsersUrl}/users")
    .postData(User)
    .header(HttpHeaderNames.ContentType, ContentType)
    .asString

  // Login user
  val tokenRes = Http(s"${UsersUrl}/tokens")
    .postData(User)
    .header(HttpHeaderNames.ContentType, ContentType)
    .asString
    .body

  val tokenCursor = parse(tokenRes).getOrElse(Json.Null).hcursor
  val token = tokenCursor.downField("token").as[String].getOrElse("")

  // Register client
  val clientLocation = Http(s"${ClientsUrl}/clients")
    .postData(Client)
    .header(HttpHeaderNames.Authorization, token)
    .header(HttpHeaderNames.ContentType, ContentType)
    .asString
    .headers.get("Location").get(0)

  val clientId = clientLocation.split("/")(2)

  // Get client key
  val clientRes = Http(s"${ClientsUrl}/clients/${clientId}")
    .header(HttpHeaderNames.Authorization, token)
    .header(HttpHeaderNames.ContentType, ContentType)
    .asString
    .body

  val clientCursor = parse(clientRes).getOrElse(Json.Null).hcursor
  val clientKey = clientCursor.downField("key").as[String].getOrElse("")

  // Register channel
  val chanLocation = Http(s"${ClientsUrl}/channels")
    .postData(Channel)
    .header(HttpHeaderNames.Authorization, token)
    .header(HttpHeaderNames.ContentType, ContentType)
    .asString
    .headers.get("Location").get(0)

  val chanId = chanLocation.split("/")(2)

  // Connect client to channel
  Http(s"${ClientsUrl}/channels/${chanId}/clients/${clientId}")
    .method("PUT")
    .header(HttpHeaderNames.Authorization, token)
    .asString

  // Prepare testing scenario
  val httpProtocol = http
    .baseURL(HttpAdapterUrl)
    .inferHtmlResources()
    .acceptHeader("*/*")
    .contentTypeHeader("application/json; charset=utf-8")
    .userAgentHeader("curl/7.54.0")

  val scn = scenario("PublishMessage")
    .exec(http("PublishMessageRequest")
      .post(s"/channels/${chanId}/messages")
      .header(HttpHeaderNames.ContentType, "application/senml+json")
      .header(HttpHeaderNames.Authorization, clientKey)
      .body(StringBody(Message))
      .check(status.is(202)))

  setUp(
    scn.inject(
      constantUsersPerSec(RequestsPerSecond.toDouble) during (15 second))).protocols(httpProtocol)
}

object PublishSimulation {
  val ContentType = "application/json"
  val User = """{"email":"john.doe@email.com", "password":"123"}"""
  val Client = """{"type":"device", "name":"weio"}"""
  val Channel = """{"name":"mychan"}"""
  val Message = """[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]"""
}
