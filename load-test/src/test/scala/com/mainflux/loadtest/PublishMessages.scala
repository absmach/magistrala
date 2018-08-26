/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package com.mainflux.loadtest

import io.circe._
import io.circe.parser._
import io.gatling.http.Predef._
import scalaj.http.Http

abstract class PublishMessages extends TestCase {
  def makeThing(token: String): (String, String) = {
    val thing = """{"type":"device", "name":"weio"}"""

    val id = Http(s"$ThingsURL/things")
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
}
