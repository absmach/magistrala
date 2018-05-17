package com.mainflux.loadtest

object Constants {
  val UsersURL: String = System.getProperty("users", "http://localhost:8180")
  val ThingsURL: String = System.getProperty("things", "http://localhost:8182")
  val HttpAdapterURL: String = System.getProperty("http", "http://localhost:8185")
  val RequestsPerSecond: Double = Integer.getInteger("requests", 100).toDouble
}