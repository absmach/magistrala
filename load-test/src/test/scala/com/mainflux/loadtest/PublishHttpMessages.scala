/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package com.mainflux.loadtest

import scala.concurrent.duration._

import io.gatling.core.Predef._
import io.gatling.http.Predef._
import io.gatling.http.request.builder.HttpRequestBuilder.toActionBuilder

final class PublishHttpMessages extends PublishMessages {
  override def prepareAndExecute(): SetUp = {
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

    connect(channelID, thingID, token)

    val scn = scenario("publish message over HTTP")
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
