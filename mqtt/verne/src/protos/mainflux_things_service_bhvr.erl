%%%-------------------------------------------------------------------
%% @doc Behaviour to implement for grpc service mainflux.ThingsService.
%% @end
%%%-------------------------------------------------------------------

%% this module was generated on 2019-11-04T09:57:59+00:00 and should not be modified manually

-module(mainflux_things_service_bhvr).

%% @doc Unary RPC
-callback can_access_by_key(ctx:ctx(), internal_pb:access_by_key_req()) ->
    {ok, internal_pb:thing_id(), ctx:ctx()} | grpcbox_stream:grpc_error_response().

%% @doc Unary RPC
-callback can_access_by_id(ctx:ctx(), internal_pb:access_by_id_req()) ->
    {ok, internal_pb:empty(), ctx:ctx()} | grpcbox_stream:grpc_error_response().

%% @doc Unary RPC
-callback identify(ctx:ctx(), internal_pb:token()) ->
    {ok, internal_pb:thing_id(), ctx:ctx()} | grpcbox_stream:grpc_error_response().

