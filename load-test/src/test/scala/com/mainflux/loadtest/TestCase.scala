/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package com.mainflux.loadtest

import io.circe._
import io.circe.parser._
import io.gatling.core.Predef._
import io.gatling.http.Predef._
import io.gatling.http.protocol.HttpProtocol
import scalaj.http.Http

trait TestCase extends Simulation {
  protected lazy val UsersURL: String = System.getProperty("users", "http://localhost:8180")
  protected lazy val ThingsURL: String = System.getProperty("things", "http://localhost:8182")
  protected lazy val RequestsPerSecond: Double = Integer.getInteger("requests", 100).toDouble

  protected val jsonType: String = "application/json"

  def authenticate(): String = {
    val user = """{"email":"john.doe@email.com", "password":"123"}"""
    val headerName = HttpHeaderNames.ContentType
    val contentType = jsonType

    Http(s"$UsersURL/users")
      .postData(user)
      .header(headerName, contentType)
      .asString

    val res = Http(s"$UsersURL/tokens")
      .postData(user)
      .header(headerName, contentType)
      .asString
      .body

    val cursor = parse(res).getOrElse(Json.Null).hcursor
    cursor.downField("token").as[String].getOrElse("")
  }

  def httpProtocol(url: String): HttpProtocol = http
    .baseURL(url)
    .inferHtmlResources()
    .acceptHeader("*/*")
    .contentTypeHeader(jsonType)
    .userAgentHeader("curl/7.54.0")
    .build

  def wsProtocol(url: String): HttpProtocol = http
    .baseURL(s"http://$url")
    .inferHtmlResources()
    .acceptHeader("*/*")
    .userAgentHeader("Gatling2")
    .wsBaseURL(s"ws://$url")
    .build

  def prepareAndExecute(): SetUp

  prepareAndExecute()
}
