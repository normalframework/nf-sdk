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
proto.normalgw.platform = require('./config_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.platform.ConfigurationClient =
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
proto.normalgw.platform.ConfigurationPromiseClient =
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
 *   !proto.normalgw.platform.GetSystemInformationRequest,
 *   !proto.normalgw.platform.GetSystemInformationReply>}
 */
const methodDescriptor_Configuration_GetSystemInformation = new grpc.web.MethodDescriptor(
  '/normalgw.platform.Configuration/GetSystemInformation',
  grpc.web.MethodType.UNARY,
  proto.normalgw.platform.GetSystemInformationRequest,
  proto.normalgw.platform.GetSystemInformationReply,
  /**
   * @param {!proto.normalgw.platform.GetSystemInformationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.platform.GetSystemInformationReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.platform.GetSystemInformationRequest,
 *   !proto.normalgw.platform.GetSystemInformationReply>}
 */
const methodInfo_Configuration_GetSystemInformation = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.platform.GetSystemInformationReply,
  /**
   * @param {!proto.normalgw.platform.GetSystemInformationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.platform.GetSystemInformationReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.platform.GetSystemInformationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.platform.GetSystemInformationReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.platform.GetSystemInformationReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.platform.ConfigurationClient.prototype.getSystemInformation =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.platform.Configuration/GetSystemInformation',
      request,
      metadata || {},
      methodDescriptor_Configuration_GetSystemInformation,
      callback);
};


/**
 * @param {!proto.normalgw.platform.GetSystemInformationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.platform.GetSystemInformationReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.platform.ConfigurationPromiseClient.prototype.getSystemInformation =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.platform.Configuration/GetSystemInformation',
      request,
      metadata || {},
      methodDescriptor_Configuration_GetSystemInformation);
};


module.exports = proto.normalgw.platform;

