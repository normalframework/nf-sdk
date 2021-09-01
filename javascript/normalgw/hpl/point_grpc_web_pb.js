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
const proto = {};
proto.normalgw = {};
proto.normalgw.hpl = require('./point_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.hpl.PointManagerClient =
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
proto.normalgw.hpl.PointManagerPromiseClient =
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
 *   !proto.normalgw.hpl.GetPointsRequest,
 *   !proto.normalgw.hpl.GetPointsReply>}
 */
const methodDescriptor_PointManager_GetPoints = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/GetPoints',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.GetPointsRequest,
  proto.normalgw.hpl.GetPointsReply,
  /**
   * @param {!proto.normalgw.hpl.GetPointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetPointsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.GetPointsRequest,
 *   !proto.normalgw.hpl.GetPointsReply>}
 */
const methodInfo_PointManager_GetPoints = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.GetPointsReply,
  /**
   * @param {!proto.normalgw.hpl.GetPointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetPointsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.GetPointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.GetPointsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.GetPointsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.getPoints =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetPoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetPoints,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.GetPointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.GetPointsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.getPoints =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetPoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetPoints);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.GetPointsByIdRequest,
 *   !proto.normalgw.hpl.GetPointsReply>}
 */
const methodDescriptor_PointManager_GetPointsById = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/GetPointsById',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.GetPointsByIdRequest,
  proto.normalgw.hpl.GetPointsReply,
  /**
   * @param {!proto.normalgw.hpl.GetPointsByIdRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetPointsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.GetPointsByIdRequest,
 *   !proto.normalgw.hpl.GetPointsReply>}
 */
const methodInfo_PointManager_GetPointsById = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.GetPointsReply,
  /**
   * @param {!proto.normalgw.hpl.GetPointsByIdRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetPointsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.GetPointsByIdRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.GetPointsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.GetPointsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.getPointsById =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetPointsById',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetPointsById,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.GetPointsByIdRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.GetPointsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.getPointsById =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetPointsById',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetPointsById);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.GetDistinctAttrsRequest,
 *   !proto.normalgw.hpl.GetDistinctAttrsReply>}
 */
const methodDescriptor_PointManager_GetDistinctAttrs = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/GetDistinctAttrs',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.GetDistinctAttrsRequest,
  proto.normalgw.hpl.GetDistinctAttrsReply,
  /**
   * @param {!proto.normalgw.hpl.GetDistinctAttrsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetDistinctAttrsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.GetDistinctAttrsRequest,
 *   !proto.normalgw.hpl.GetDistinctAttrsReply>}
 */
const methodInfo_PointManager_GetDistinctAttrs = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.GetDistinctAttrsReply,
  /**
   * @param {!proto.normalgw.hpl.GetDistinctAttrsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetDistinctAttrsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.GetDistinctAttrsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.GetDistinctAttrsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.GetDistinctAttrsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.getDistinctAttrs =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetDistinctAttrs',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetDistinctAttrs,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.GetDistinctAttrsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.GetDistinctAttrsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.getDistinctAttrs =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetDistinctAttrs',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetDistinctAttrs);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.GetDataRequest,
 *   !proto.normalgw.hpl.GetDataReply>}
 */
const methodDescriptor_PointManager_GetData = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/GetData',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.GetDataRequest,
  proto.normalgw.hpl.GetDataReply,
  /**
   * @param {!proto.normalgw.hpl.GetDataRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetDataReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.GetDataRequest,
 *   !proto.normalgw.hpl.GetDataReply>}
 */
const methodInfo_PointManager_GetData = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.GetDataReply,
  /**
   * @param {!proto.normalgw.hpl.GetDataRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.GetDataReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.GetDataRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.GetDataReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.GetDataReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.getData =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetData',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetData,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.GetDataRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.GetDataReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.getData =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/GetData',
      request,
      metadata || {},
      methodDescriptor_PointManager_GetData);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.AddPointsDataRequest,
 *   !proto.normalgw.hpl.AddPointsDataReply>}
 */
const methodDescriptor_PointManager_AddPointsData = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/AddPointsData',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.AddPointsDataRequest,
  proto.normalgw.hpl.AddPointsDataReply,
  /**
   * @param {!proto.normalgw.hpl.AddPointsDataRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.AddPointsDataReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.AddPointsDataRequest,
 *   !proto.normalgw.hpl.AddPointsDataReply>}
 */
const methodInfo_PointManager_AddPointsData = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.AddPointsDataReply,
  /**
   * @param {!proto.normalgw.hpl.AddPointsDataRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.AddPointsDataReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.AddPointsDataRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.AddPointsDataReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.AddPointsDataReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.addPointsData =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/AddPointsData',
      request,
      metadata || {},
      methodDescriptor_PointManager_AddPointsData,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.AddPointsDataRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.AddPointsDataReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.addPointsData =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/AddPointsData',
      request,
      metadata || {},
      methodDescriptor_PointManager_AddPointsData);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.UpdatePointsRequest,
 *   !proto.normalgw.hpl.UpdatePointsReply>}
 */
const methodDescriptor_PointManager_UpdatePoints = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/UpdatePoints',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.UpdatePointsRequest,
  proto.normalgw.hpl.UpdatePointsReply,
  /**
   * @param {!proto.normalgw.hpl.UpdatePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.UpdatePointsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.UpdatePointsRequest,
 *   !proto.normalgw.hpl.UpdatePointsReply>}
 */
const methodInfo_PointManager_UpdatePoints = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.UpdatePointsReply,
  /**
   * @param {!proto.normalgw.hpl.UpdatePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.UpdatePointsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.UpdatePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.UpdatePointsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.UpdatePointsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.updatePoints =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/UpdatePoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_UpdatePoints,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.UpdatePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.UpdatePointsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.updatePoints =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/UpdatePoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_UpdatePoints);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.RenamePointRequest,
 *   !proto.normalgw.hpl.RenamePointReply>}
 */
const methodDescriptor_PointManager_RenamePoint = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/RenamePoint',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.RenamePointRequest,
  proto.normalgw.hpl.RenamePointReply,
  /**
   * @param {!proto.normalgw.hpl.RenamePointRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.RenamePointReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.RenamePointRequest,
 *   !proto.normalgw.hpl.RenamePointReply>}
 */
const methodInfo_PointManager_RenamePoint = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.RenamePointReply,
  /**
   * @param {!proto.normalgw.hpl.RenamePointRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.RenamePointReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.RenamePointRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.RenamePointReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.RenamePointReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.renamePoint =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/RenamePoint',
      request,
      metadata || {},
      methodDescriptor_PointManager_RenamePoint,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.RenamePointRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.RenamePointReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.renamePoint =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/RenamePoint',
      request,
      metadata || {},
      methodDescriptor_PointManager_RenamePoint);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.DeletePointsRequest,
 *   !proto.normalgw.hpl.DeletePointsReply>}
 */
const methodDescriptor_PointManager_DeletePoints = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/DeletePoints',
  grpc.web.MethodType.UNARY,
  proto.normalgw.hpl.DeletePointsRequest,
  proto.normalgw.hpl.DeletePointsReply,
  /**
   * @param {!proto.normalgw.hpl.DeletePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.DeletePointsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.DeletePointsRequest,
 *   !proto.normalgw.hpl.DeletePointsReply>}
 */
const methodInfo_PointManager_DeletePoints = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.DeletePointsReply,
  /**
   * @param {!proto.normalgw.hpl.DeletePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.DeletePointsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.DeletePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.hpl.DeletePointsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.DeletePointsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.deletePoints =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.hpl.PointManager/DeletePoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_DeletePoints,
      callback);
};


/**
 * @param {!proto.normalgw.hpl.DeletePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.hpl.DeletePointsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.deletePoints =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.hpl.PointManager/DeletePoints',
      request,
      metadata || {},
      methodDescriptor_PointManager_DeletePoints);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.ObservePointsUpdatesRequest,
 *   !proto.normalgw.hpl.ObservePointsUpdatesReply>}
 */
const methodDescriptor_PointManager_ObservePointsUpdates = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/ObservePointsUpdates',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.hpl.ObservePointsUpdatesRequest,
  proto.normalgw.hpl.ObservePointsUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObservePointsUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObservePointsUpdatesReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.ObservePointsUpdatesRequest,
 *   !proto.normalgw.hpl.ObservePointsUpdatesReply>}
 */
const methodInfo_PointManager_ObservePointsUpdates = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.ObservePointsUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObservePointsUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObservePointsUpdatesReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.ObservePointsUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObservePointsUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.observePointsUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObservePointsUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObservePointsUpdates);
};


/**
 * @param {!proto.normalgw.hpl.ObservePointsUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObservePointsUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.observePointsUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObservePointsUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObservePointsUpdates);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.ObserveDataUpdatesRequest,
 *   !proto.normalgw.hpl.ObserveDataUpdatesReply>}
 */
const methodDescriptor_PointManager_ObserveDataUpdates = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/ObserveDataUpdates',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.hpl.ObserveDataUpdatesRequest,
  proto.normalgw.hpl.ObserveDataUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveDataUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveDataUpdatesReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.ObserveDataUpdatesRequest,
 *   !proto.normalgw.hpl.ObserveDataUpdatesReply>}
 */
const methodInfo_PointManager_ObserveDataUpdates = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.ObserveDataUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveDataUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveDataUpdatesReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.ObserveDataUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveDataUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.observeDataUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObserveDataUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObserveDataUpdates);
};


/**
 * @param {!proto.normalgw.hpl.ObserveDataUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveDataUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.observeDataUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObserveDataUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObserveDataUpdates);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.hpl.ObserveErrorUpdatesRequest,
 *   !proto.normalgw.hpl.ObserveErrorUpdatesReply>}
 */
const methodDescriptor_PointManager_ObserveErrorUpdates = new grpc.web.MethodDescriptor(
  '/normalgw.hpl.PointManager/ObserveErrorUpdates',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.hpl.ObserveErrorUpdatesRequest,
  proto.normalgw.hpl.ObserveErrorUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveErrorUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveErrorUpdatesReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.hpl.ObserveErrorUpdatesRequest,
 *   !proto.normalgw.hpl.ObserveErrorUpdatesReply>}
 */
const methodInfo_PointManager_ObserveErrorUpdates = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.hpl.ObserveErrorUpdatesReply,
  /**
   * @param {!proto.normalgw.hpl.ObserveErrorUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.hpl.ObserveErrorUpdatesReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.hpl.ObserveErrorUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveErrorUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerClient.prototype.observeErrorUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObserveErrorUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObserveErrorUpdates);
};


/**
 * @param {!proto.normalgw.hpl.ObserveErrorUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.hpl.ObserveErrorUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.hpl.PointManagerPromiseClient.prototype.observeErrorUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.hpl.PointManager/ObserveErrorUpdates',
      request,
      metadata || {},
      methodDescriptor_PointManager_ObserveErrorUpdates);
};


module.exports = proto.normalgw.hpl;

