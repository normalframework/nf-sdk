/**
 * @fileoverview gRPC-Web generated client stub for normalgw.platform
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');

const proto = {};
proto.normalgw = {};
proto.normalgw.platform = require('./logs_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.platform.LogsClient =
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
proto.normalgw.platform.LogsPromiseClient =
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
 *   !proto.normalgw.platform.GetLogRequest,
 *   !proto.normalgw.platform.LogMessage>}
 */
const methodDescriptor_Logs_GetLogs = new grpc.web.MethodDescriptor(
  '/normalgw.platform.Logs/GetLogs',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.platform.GetLogRequest,
  proto.normalgw.platform.LogMessage,
  /**
   * @param {!proto.normalgw.platform.GetLogRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.platform.LogMessage.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.platform.GetLogRequest,
 *   !proto.normalgw.platform.LogMessage>}
 */
const methodInfo_Logs_GetLogs = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.platform.LogMessage,
  /**
   * @param {!proto.normalgw.platform.GetLogRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.platform.LogMessage.deserializeBinary
);


/**
 * @param {!proto.normalgw.platform.GetLogRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.platform.LogMessage>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.platform.LogsClient.prototype.getLogs =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.platform.Logs/GetLogs',
      request,
      metadata || {},
      methodDescriptor_Logs_GetLogs);
};


/**
 * @param {!proto.normalgw.platform.GetLogRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.platform.LogMessage>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.platform.LogsPromiseClient.prototype.getLogs =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.platform.Logs/GetLogs',
      request,
      metadata || {},
      methodDescriptor_Logs_GetLogs);
};


module.exports = proto.normalgw.platform;

