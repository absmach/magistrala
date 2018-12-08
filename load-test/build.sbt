enablePlugins(GatlingPlugin)

name := "load-test"
version := "0.7.0"

scalaVersion := "2.12.4"

val gatlingVersion = "2.3.1"
val circeVersion = "0.9.3"

libraryDependencies ++= Seq(
  "io.gatling.highcharts" %  "gatling-charts-highcharts" % gatlingVersion,
  "io.gatling"            %  "gatling-test-framework"    % gatlingVersion,
  "org.scalaj"            %% "scalaj-http"               % "2.3.0",
  "io.circe"              %% "circe-core"                % circeVersion,
  "io.circe"              %% "circe-generic"             % circeVersion,
  "io.circe"              %% "circe-parser"              % circeVersion
)
