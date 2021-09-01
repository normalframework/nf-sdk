/**
 * @fileoverview gRPC-Web generated client stub for normalgw.bacnet
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');


var google_api_annotations_pb = require('../../google/api/annotations_pb.js')

var normalgw_bacnet_bacenum_pb = require('../../normalgw/bacnet/bacenum_pb.js')
const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./bacnet_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.BacnetClient =
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
proto.normalgw.bacnet.BacnetPromiseClient =
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
 *   !proto.normalgw.bacnet.WhoIsRequest,
 *   !proto.normalgw.bacnet.WhoIsReply>}
 */
const methodDescriptor_Bacnet_WhoIs = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/WhoIs',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.bacnet.WhoIsRequest,
  proto.normalgw.bacnet.WhoIsReply,
  /**
   * @param {!proto.normalgw.bacnet.WhoIsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WhoIsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.WhoIsRequest,
 *   !proto.normalgw.bacnet.WhoIsReply>}
 */
const methodInfo_Bacnet_WhoIs = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.WhoIsReply,
  /**
   * @param {!proto.normalgw.bacnet.WhoIsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WhoIsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.WhoIsRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.WhoIsReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.whoIs =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WhoIs',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WhoIs);
};


/**
 * @param {!proto.normalgw.bacnet.WhoIsRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.WhoIsReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.whoIs =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WhoIs',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WhoIs);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.WhoIsRouterRequest,
 *   !proto.normalgw.bacnet.WhoIsRouterReply>}
 */
const methodDescriptor_Bacnet_WhoIsRouter = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/WhoIsRouter',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.bacnet.WhoIsRouterRequest,
  proto.normalgw.bacnet.WhoIsRouterReply,
  /**
   * @param {!proto.normalgw.bacnet.WhoIsRouterRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WhoIsRouterReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.WhoIsRouterRequest,
 *   !proto.normalgw.bacnet.WhoIsRouterReply>}
 */
const methodInfo_Bacnet_WhoIsRouter = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.WhoIsRouterReply,
  /**
   * @param {!proto.normalgw.bacnet.WhoIsRouterRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WhoIsRouterReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.WhoIsRouterRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.WhoIsRouterReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.whoIsRouter =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WhoIsRouter',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WhoIsRouter);
};


/**
 * @param {!proto.normalgw.bacnet.WhoIsRouterRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.WhoIsRouterReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.whoIsRouter =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WhoIsRouter',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WhoIsRouter);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.ReadPropertyRequest,
 *   !proto.normalgw.bacnet.ReadPropertyReply>}
 */
const methodDescriptor_Bacnet_ReadProperty = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/ReadProperty',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.ReadPropertyRequest,
  proto.normalgw.bacnet.ReadPropertyReply,
  /**
   * @param {!proto.normalgw.bacnet.ReadPropertyRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ReadPropertyReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.ReadPropertyRequest,
 *   !proto.normalgw.bacnet.ReadPropertyReply>}
 */
const methodInfo_Bacnet_ReadProperty = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.ReadPropertyReply,
  /**
   * @param {!proto.normalgw.bacnet.ReadPropertyRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ReadPropertyReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.ReadPropertyRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.ReadPropertyReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ReadPropertyReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.readProperty =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/ReadProperty',
      request,
      metadata || {},
      methodDescriptor_Bacnet_ReadProperty,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.ReadPropertyRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.ReadPropertyReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.readProperty =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/ReadProperty',
      request,
      metadata || {},
      methodDescriptor_Bacnet_ReadProperty);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.ReadPropMultipleRequest,
 *   !proto.normalgw.bacnet.ReadPropMultipleReply>}
 */
const methodDescriptor_Bacnet_ReadPropMultiple = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/ReadPropMultiple',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.ReadPropMultipleRequest,
  proto.normalgw.bacnet.ReadPropMultipleReply,
  /**
   * @param {!proto.normalgw.bacnet.ReadPropMultipleRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ReadPropMultipleReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.ReadPropMultipleRequest,
 *   !proto.normalgw.bacnet.ReadPropMultipleReply>}
 */
const methodInfo_Bacnet_ReadPropMultiple = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.ReadPropMultipleReply,
  /**
   * @param {!proto.normalgw.bacnet.ReadPropMultipleRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ReadPropMultipleReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.ReadPropMultipleRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.ReadPropMultipleReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ReadPropMultipleReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.readPropMultiple =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/ReadPropMultiple',
      request,
      metadata || {},
      methodDescriptor_Bacnet_ReadPropMultiple,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.ReadPropMultipleRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.ReadPropMultipleReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.readPropMultiple =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/ReadPropMultiple',
      request,
      metadata || {},
      methodDescriptor_Bacnet_ReadPropMultiple);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.RegisterWithBbmdRequest,
 *   !proto.normalgw.bacnet.RegisterWithBbmdReply>}
 */
const methodDescriptor_Bacnet_RegisterWithBbmd = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/RegisterWithBbmd',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.RegisterWithBbmdRequest,
  proto.normalgw.bacnet.RegisterWithBbmdReply,
  /**
   * @param {!proto.normalgw.bacnet.RegisterWithBbmdRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.RegisterWithBbmdReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.RegisterWithBbmdRequest,
 *   !proto.normalgw.bacnet.RegisterWithBbmdReply>}
 */
const methodInfo_Bacnet_RegisterWithBbmd = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.RegisterWithBbmdReply,
  /**
   * @param {!proto.normalgw.bacnet.RegisterWithBbmdRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.RegisterWithBbmdReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.RegisterWithBbmdRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.RegisterWithBbmdReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.RegisterWithBbmdReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.registerWithBbmd =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/RegisterWithBbmd',
      request,
      metadata || {},
      methodDescriptor_Bacnet_RegisterWithBbmd,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.RegisterWithBbmdRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.RegisterWithBbmdReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.registerWithBbmd =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/RegisterWithBbmd',
      request,
      metadata || {},
      methodDescriptor_Bacnet_RegisterWithBbmd);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.WritePropertyRequest,
 *   !proto.normalgw.bacnet.WritePropertyReply>}
 */
const methodDescriptor_Bacnet_WriteProperty = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Bacnet/WriteProperty',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.WritePropertyRequest,
  proto.normalgw.bacnet.WritePropertyReply,
  /**
   * @param {!proto.normalgw.bacnet.WritePropertyRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WritePropertyReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.WritePropertyRequest,
 *   !proto.normalgw.bacnet.WritePropertyReply>}
 */
const methodInfo_Bacnet_WriteProperty = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.WritePropertyReply,
  /**
   * @param {!proto.normalgw.bacnet.WritePropertyRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.WritePropertyReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.WritePropertyRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.WritePropertyReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.WritePropertyReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.BacnetClient.prototype.writeProperty =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WriteProperty',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WriteProperty,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.WritePropertyRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.WritePropertyReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.BacnetPromiseClient.prototype.writeProperty =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Bacnet/WriteProperty',
      request,
      metadata || {},
      methodDescriptor_Bacnet_WriteProperty);
};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.ConfigurationClient =
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
proto.normalgw.bacnet.ConfigurationPromiseClient =
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
 *   !proto.normalgw.bacnet.GetConfigurationRequest,
 *   !proto.normalgw.bacnet.GetConfigurationReply>}
 */
const methodDescriptor_Configuration_GetConfiguration = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Configuration/GetConfiguration',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetConfigurationRequest,
  proto.normalgw.bacnet.GetConfigurationReply,
  /**
   * @param {!proto.normalgw.bacnet.GetConfigurationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetConfigurationReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetConfigurationRequest,
 *   !proto.normalgw.bacnet.GetConfigurationReply>}
 */
const methodInfo_Configuration_GetConfiguration = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetConfigurationReply,
  /**
   * @param {!proto.normalgw.bacnet.GetConfigurationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetConfigurationReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetConfigurationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetConfigurationReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetConfigurationReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ConfigurationClient.prototype.getConfiguration =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Configuration/GetConfiguration',
      request,
      metadata || {},
      methodDescriptor_Configuration_GetConfiguration,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetConfigurationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetConfigurationReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ConfigurationPromiseClient.prototype.getConfiguration =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Configuration/GetConfiguration',
      request,
      metadata || {},
      methodDescriptor_Configuration_GetConfiguration);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.SetConfigurationRequest,
 *   !proto.normalgw.bacnet.SetConfigurationReply>}
 */
const methodDescriptor_Configuration_SetConfiguration = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Configuration/SetConfiguration',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.SetConfigurationRequest,
  proto.normalgw.bacnet.SetConfigurationReply,
  /**
   * @param {!proto.normalgw.bacnet.SetConfigurationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.SetConfigurationReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.SetConfigurationRequest,
 *   !proto.normalgw.bacnet.SetConfigurationReply>}
 */
const methodInfo_Configuration_SetConfiguration = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.SetConfigurationReply,
  /**
   * @param {!proto.normalgw.bacnet.SetConfigurationRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.SetConfigurationReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.SetConfigurationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.SetConfigurationReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.SetConfigurationReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ConfigurationClient.prototype.setConfiguration =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Configuration/SetConfiguration',
      request,
      metadata || {},
      methodDescriptor_Configuration_SetConfiguration,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.SetConfigurationRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.SetConfigurationReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ConfigurationPromiseClient.prototype.setConfiguration =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Configuration/SetConfiguration',
      request,
      metadata || {},
      methodDescriptor_Configuration_SetConfiguration);
};


module.exports = proto.normalgw.bacnet;

