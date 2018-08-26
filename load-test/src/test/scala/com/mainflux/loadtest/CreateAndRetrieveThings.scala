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

final class CreateAndRetrieveThings extends TestCase {
  override def prepareAndExecute(): SetUp = {
    val token = authenticate()
    val thing = """{"type":"device", "name":"weio"}"""

    val scn = scenario("create and retrieve things")
      .exec(http("create thing")
        .post("/things")
        .header(HttpHeaderNames.ContentType, jsonType)
        .header(HttpHeaderNames.Authorization, token)
        .body(StringBody(thing))
        .check(status.is(201))
        .check(headerRegex(HttpHeaderNames.Location, "(.*)").saveAs("location")))
      .exec(http("retrieve thing")
        .get("${location}")
        .header(HttpHeaderNames.Authorization, token)
        .check(status.is(200)))

    setUp(scn.inject(constantUsersPerSec(RequestsPerSecond) during 15.seconds)).protocols(httpProtocol(ThingsURL))
  }
}
