-module(mfx_grpc).
-behaviour(gen_server).
-behaviour(poolboy_worker).

-export([
    start_link/0,
    start_link/1,
    init/1,
    handle_call/3,
    handle_cast/2,
    handle_info/2,
    terminate/2
]).

-record(state, {conn}).

init(_Args) ->
    error_logger:info_msg("mfx_grpc genserver has started (~w)~n", [self()]),
    [{_, GrpcUrl}] = ets:lookup(mfx_cfg, grpc_url),
    {ok, {_, _, GrpcHost, GrpcPort, _, _}} = http_uri:parse(GrpcUrl),
    error_logger:info_msg("grpc host: ~p,  port: ~p", [GrpcHost, GrpcPort]),
    {ok, GrpcConn} = grpc_client:connect(tcp, GrpcHost, GrpcPort),
    {ok, #state{conn = GrpcConn}}.

start_link() ->
    gen_server:start_link({local, ?MODULE}, ?MODULE, [], []).

start_link(Args) ->
    gen_server:start_link(?MODULE, Args, []).

handle_call({identify, Message}, _From, #state{conn = GrpcConn} = State) ->
    error_logger:info_msg("mfx_grpc message: ~p", [Message]),
    {Status, Result} = internal_client:'IdentifyThing'(GrpcConn, Message, []),
    case Status of
      ok ->
          #{
              grpc_status := 0,
              headers := #{<<":status">> := <<"200">>},
              http_status := HttpStatus,
              result :=
                  #{value := ThingId},
              status_message := <<>>,
              trailers := #{<<"grpc-status">> := <<"0">>}
          } = Result,

          case HttpStatus of
              200 ->
                  {reply, {ok, list_to_binary(ThingId)}, State};
              _ ->
                  {reply, {error, HttpStatus}, error}
          end;
      _ ->
          {reply, {error, Status}, State}
  end;
handle_call({can_access_by_id, Message}, _From, #state{conn = GrpcConn} = State) ->
  error_logger:info_msg("mfx_grpc message: ~p", [Message]),
  {Status, Result} = internal_client:'CanAccessByID'(GrpcConn, Message, []),
  case Status of
    ok ->
      #{
        grpc_status := 0,
        headers := #{
          <<":status">> := <<"200">>,
          <<"content-type">> := <<"application/grpc+proto">>
        },
        http_status := HttpStatus,
        result := #{},
        status_message := <<>>,
        trailers := #{
          <<"grpc-message">> := <<>>,
          <<"grpc-status">> := <<"0">>}
      } = Result,

      case HttpStatus of
          200 ->
              {reply, ok, State};
          _ ->
              {reply, {error, HttpStatus}, State}
      end;
      
    _ ->
        {reply, {error, Status}, State}
  end.

handle_cast(_Request, State) ->
    {noreply, State}.

handle_info(_Info, State) ->
    {noreply, State}.

terminate(_Reason, _State) ->
    [].

