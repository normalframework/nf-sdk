/**
 * @fileoverview gRPC-Web generated client stub for normalgw.hpl
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');


var google_api_annotations_pb = require('../../google/api/annotations_pb.js')

var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js')

var google_protobuf_duration_pb = require('google-protobuf/google/protobuf/duration_pb.js')

var google_protobuf_any_pb = require('google-protobuf/google/protobuf/any_pb.js')

var normalgw_hpl_point_pb = require('../../normalgw/hpl/point_pb.js')
const proto = {};
proto.normalgw = {};
proto.normalgw.hpl = require('./command_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.hpl.CommandManagerClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.hpl.CommandManagerPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.StartCommandRequest,
 *   !proto.normalgw.hpl.StartCommandReply>}
 */
const methodDescriptor_CommandManager_StartCommand = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.CommandManager/StartCommand',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.StartCommandRequest,
  proto.normalgw.hpl.StartCommandReply,
  /**
   * @param {!proto.normalgw.hpl.StartCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.StartCommandReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.StartCommandRequest,
 *   !proto.normalgw.hpl.StartCommandReply>}
 */
const methodInfo_CommandManager_StartCommand = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.StartCommandReply,
  /**
   * @param {!proto.normalgw.hpl.StartCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.StartCommandReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.StartCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.StartCommandReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.StartCommandReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerClient.prototype.startCommand =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/StartCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_StartCommand,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.StartCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.StartCommandReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.CommandManagerPromiseClient.prototype.startCommand =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/StartCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_StartCommand);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.GetCommandsRequest,
 *   !proto.normalgw.hpl.GetCommandsReply>}
 */
const methodDescriptor_CommandManager_GetCommands = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.CommandManager/GetCommands',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.GetCommandsRequest,
  proto.normalgw.hpl.GetCommandsReply,
  /**
   * @param {!proto.normalgw.hpl.GetCommandsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetCommandsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.GetCommandsRequest,
 *   !proto.normalgw.hpl.GetCommandsReply>}
 */
const methodInfo_CommandManager_GetCommands = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.GetCommandsReply,
  /**
   * @param {!proto.normalgw.hpl.GetCommandsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetCommandsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.GetCommandsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.GetCommandsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.GetCommandsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerClient.prototype.getCommands =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/GetCommands',
      request,
      metadata || {},
      methodDescriptor_CommandManager_GetCommands,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.GetCommandsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.GetCommandsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.CommandManagerPromiseClient.prototype.getCommands =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/GetCommands',
      request,
      metadata || {},
      methodDescriptor_CommandManager_GetCommands);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.CancelCommandRequest,
 *   !proto.normalgw.hpl.CancelCommandReply>}
 */
const methodDescriptor_CommandManager_CancelCommand = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.CommandManager/CancelCommand',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.CancelCommandRequest,
  proto.normalgw.hpl.CancelCommandReply,
  /**
   * @param {!proto.normalgw.hpl.CancelCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.CancelCommandReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.CancelCommandRequest,
 *   !proto.normalgw.hpl.CancelCommandReply>}
 */
const methodInfo_CommandManager_CancelCommand = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.CancelCommandReply,
  /**
   * @param {!proto.normalgw.hpl.CancelCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.CancelCommandReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.CancelCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.CancelCommandReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.CancelCommandReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerClient.prototype.cancelCommand =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/CancelCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_CancelCommand,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.CancelCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.CancelCommandReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.CommandManagerPromiseClient.prototype.cancelCommand =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/CancelCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_CancelCommand);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.ExtendCommandRequest,
 *   !proto.normalgw.hpl.ExtendCommandReply>}
 */
const methodDescriptor_CommandManager_ExtendCommand = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.CommandManager/ExtendCommand',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.ExtendCommandRequest,
  proto.normalgw.hpl.ExtendCommandReply,
  /**
   * @param {!proto.normalgw.hpl.ExtendCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ExtendCommandReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.ExtendCommandRequest,
 *   !proto.normalgw.hpl.ExtendCommandReply>}
 */
const methodInfo_CommandManager_ExtendCommand = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.ExtendCommandReply,
  /**
   * @param {!proto.normalgw.hpl.ExtendCommandRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ExtendCommandReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.ExtendCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.ExtendCommandReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ExtendCommandReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerClient.prototype.extendCommand =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/ExtendCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_ExtendCommand,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.ExtendCommandRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.ExtendCommandReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.CommandManagerPromiseClient.prototype.extendCommand =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.CommandManager/ExtendCommand',
      request,
      metadata || {},
      methodDescriptor_CommandManager_ExtendCommand);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.ObserveRawCommandActionsRequest,
 *   !proto.normalgw.hpl.ObserveRawCommandActionsReply>}
 */
const methodDescriptor_CommandManager_ObserveRawCommandActions = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.CommandManager/ObserveRawCommandActions',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.hpl.ObserveRawCommandActionsRequest,
  proto.normalgw.hpl.ObserveRawCommandActionsReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveRawCommandActionsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveRawCommandActionsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.ObserveRawCommandActionsRequest,
 *   !proto.normalgw.hpl.ObserveRawCommandActionsReply>}
 */
const methodInfo_CommandManager_ObserveRawCommandActions = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.ObserveRawCommandActionsReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveRawCommandActionsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveRawCommandActionsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.ObserveRawCommandActionsRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveRawCommandActionsReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerClient.prototype.observeRawCommandActions =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.CommandManager/ObserveRawCommandActions',
      request,
      metadata || {},
      methodDescriptor_CommandManager_ObserveRawCommandActions);
};


/**
 * @param {!proto.normalgw.hpl.ObserveRawCommandActionsRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveRawCommandActionsReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.CommandManagerPromiseClient.prototype.observeRawCommandActions =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.CommandManager/ObserveRawCommandActions',
      request,
      metadata || {},
      methodDescriptor_CommandManager_ObserveRawCommandActions);
};


module.exports = proto.normalgw.hpl;

