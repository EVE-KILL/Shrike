/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars*/
import * as $protobufModule from "protobufjs/minimal.js";

const $protobuf = $protobufModule.default ?? $protobufModule;

// Common aliases
const $Reader = $protobuf.Reader, $util = $protobuf.util;

// Exported root namespace
const $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

export const esf = $root.esf = (() => {

    /**
     * Namespace esf.
     * @exports esf
     * @namespace
     */
    const esf = {};

    esf.TypeDogma = (function() {

        /**
         * Properties of a TypeDogma.
         * @memberof esf
         * @interface ITypeDogma
         * @property {Object.<string,esf.TypeDogma.ITypeDogmaEntry>|null} [entries] TypeDogma entries
         */

        /**
         * Constructs a new TypeDogma.
         * @memberof esf
         * @classdesc Represents a TypeDogma.
         * @implements ITypeDogma
         * @constructor
         * @param {esf.ITypeDogma=} [properties] Properties to set
         */
        function TypeDogma(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * TypeDogma entries.
         * @member {Object.<string,esf.TypeDogma.ITypeDogmaEntry>} entries
         * @memberof esf.TypeDogma
         * @instance
         */
        TypeDogma.prototype.entries = $util.emptyObject;

        /**
         * Decodes a TypeDogma message from the specified reader or buffer.
         * @function decode
         * @memberof esf.TypeDogma
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.TypeDogma} TypeDogma
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        TypeDogma.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.TypeDogma(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.TypeDogma.TypeDogmaEntry.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a TypeDogma message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.TypeDogma
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.TypeDogma} TypeDogma
         */
        TypeDogma.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.TypeDogma)
                return object;
            let message = new $root.esf.TypeDogma();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.TypeDogma.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.TypeDogma.entries: object expected");
                    message.entries[keys[i]] = $root.esf.TypeDogma.TypeDogmaEntry.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a TypeDogma message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.TypeDogma
         * @static
         * @param {esf.TypeDogma} message TypeDogma
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        TypeDogma.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.TypeDogma.TypeDogmaEntry.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this TypeDogma to JSON.
         * @function toJSON
         * @memberof esf.TypeDogma
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        TypeDogma.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for TypeDogma
         * @function getTypeUrl
         * @memberof esf.TypeDogma
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        TypeDogma.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.TypeDogma";
        };

        TypeDogma.TypeDogmaEntry = (function() {

            /**
             * Properties of a TypeDogmaEntry.
             * @memberof esf.TypeDogma
             * @interface ITypeDogmaEntry
             * @property {Array.<esf.TypeDogma.TypeDogmaEntry.IDogmaAttributes>|null} [dogmaAttributes] TypeDogmaEntry dogmaAttributes
             * @property {Array.<esf.TypeDogma.TypeDogmaEntry.IDogmaEffects>|null} [dogmaEffects] TypeDogmaEntry dogmaEffects
             */

            /**
             * Constructs a new TypeDogmaEntry.
             * @memberof esf.TypeDogma
             * @classdesc Represents a TypeDogmaEntry.
             * @implements ITypeDogmaEntry
             * @constructor
             * @param {esf.TypeDogma.ITypeDogmaEntry=} [properties] Properties to set
             */
            function TypeDogmaEntry(properties) {
                this.dogmaAttributes = [];
                this.dogmaEffects = [];
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * TypeDogmaEntry dogmaAttributes.
             * @member {Array.<esf.TypeDogma.TypeDogmaEntry.IDogmaAttributes>} dogmaAttributes
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @instance
             */
            TypeDogmaEntry.prototype.dogmaAttributes = $util.emptyArray;

            /**
             * TypeDogmaEntry dogmaEffects.
             * @member {Array.<esf.TypeDogma.TypeDogmaEntry.IDogmaEffects>} dogmaEffects
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @instance
             */
            TypeDogmaEntry.prototype.dogmaEffects = $util.emptyArray;

            /**
             * Decodes a TypeDogmaEntry message from the specified reader or buffer.
             * @function decode
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.TypeDogma.TypeDogmaEntry} TypeDogmaEntry
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TypeDogmaEntry.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.TypeDogma.TypeDogmaEntry();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            if (!(message.dogmaAttributes && message.dogmaAttributes.length))
                                message.dogmaAttributes = [];
                            message.dogmaAttributes.push($root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes.decode(reader, reader.uint32()));
                            break;
                        }
                    case 2: {
                            if (!(message.dogmaEffects && message.dogmaEffects.length))
                                message.dogmaEffects = [];
                            message.dogmaEffects.push($root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects.decode(reader, reader.uint32()));
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Creates a TypeDogmaEntry message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.TypeDogma.TypeDogmaEntry} TypeDogmaEntry
             */
            TypeDogmaEntry.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.TypeDogma.TypeDogmaEntry)
                    return object;
                let message = new $root.esf.TypeDogma.TypeDogmaEntry();
                if (object.dogmaAttributes) {
                    if (!Array.isArray(object.dogmaAttributes))
                        throw TypeError(".esf.TypeDogma.TypeDogmaEntry.dogmaAttributes: array expected");
                    message.dogmaAttributes = [];
                    for (let i = 0; i < object.dogmaAttributes.length; ++i) {
                        if (typeof object.dogmaAttributes[i] !== "object")
                            throw TypeError(".esf.TypeDogma.TypeDogmaEntry.dogmaAttributes: object expected");
                        message.dogmaAttributes[i] = $root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes.fromObject(object.dogmaAttributes[i]);
                    }
                }
                if (object.dogmaEffects) {
                    if (!Array.isArray(object.dogmaEffects))
                        throw TypeError(".esf.TypeDogma.TypeDogmaEntry.dogmaEffects: array expected");
                    message.dogmaEffects = [];
                    for (let i = 0; i < object.dogmaEffects.length; ++i) {
                        if (typeof object.dogmaEffects[i] !== "object")
                            throw TypeError(".esf.TypeDogma.TypeDogmaEntry.dogmaEffects: object expected");
                        message.dogmaEffects[i] = $root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects.fromObject(object.dogmaEffects[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a TypeDogmaEntry message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @static
             * @param {esf.TypeDogma.TypeDogmaEntry} message TypeDogmaEntry
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            TypeDogmaEntry.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.arrays || options.defaults) {
                    object.dogmaAttributes = [];
                    object.dogmaEffects = [];
                }
                if (message.dogmaAttributes && message.dogmaAttributes.length) {
                    object.dogmaAttributes = [];
                    for (let j = 0; j < message.dogmaAttributes.length; ++j)
                        object.dogmaAttributes[j] = $root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes.toObject(message.dogmaAttributes[j], options);
                }
                if (message.dogmaEffects && message.dogmaEffects.length) {
                    object.dogmaEffects = [];
                    for (let j = 0; j < message.dogmaEffects.length; ++j)
                        object.dogmaEffects[j] = $root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects.toObject(message.dogmaEffects[j], options);
                }
                return object;
            };

            /**
             * Converts this TypeDogmaEntry to JSON.
             * @function toJSON
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            TypeDogmaEntry.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for TypeDogmaEntry
             * @function getTypeUrl
             * @memberof esf.TypeDogma.TypeDogmaEntry
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            TypeDogmaEntry.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.TypeDogma.TypeDogmaEntry";
            };

            TypeDogmaEntry.DogmaAttributes = (function() {

                /**
                 * Properties of a DogmaAttributes.
                 * @memberof esf.TypeDogma.TypeDogmaEntry
                 * @interface IDogmaAttributes
                 * @property {number} attributeID DogmaAttributes attributeID
                 * @property {number} value DogmaAttributes value
                 */

                /**
                 * Constructs a new DogmaAttributes.
                 * @memberof esf.TypeDogma.TypeDogmaEntry
                 * @classdesc Represents a DogmaAttributes.
                 * @implements IDogmaAttributes
                 * @constructor
                 * @param {esf.TypeDogma.TypeDogmaEntry.IDogmaAttributes=} [properties] Properties to set
                 */
                function DogmaAttributes(properties) {
                    if (properties)
                        for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * DogmaAttributes attributeID.
                 * @member {number} attributeID
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @instance
                 */
                DogmaAttributes.prototype.attributeID = 0;

                /**
                 * DogmaAttributes value.
                 * @member {number} value
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @instance
                 */
                DogmaAttributes.prototype.value = 0;

                /**
                 * Decodes a DogmaAttributes message from the specified reader or buffer.
                 * @function decode
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {esf.TypeDogma.TypeDogmaEntry.DogmaAttributes} DogmaAttributes
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                DogmaAttributes.decode = function decode(reader, length, error) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes();
                    while (reader.pos < end) {
                        let tag = reader.uint32();
                        if (tag === error)
                            break;
                        switch (tag >>> 3) {
                        case 1: {
                                message.attributeID = reader.int32();
                                break;
                            }
                        case 2: {
                                message.value = reader.float();
                                break;
                            }
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    if (!message.hasOwnProperty("attributeID"))
                        throw $util.ProtocolError("missing required 'attributeID'", { instance: message });
                    if (!message.hasOwnProperty("value"))
                        throw $util.ProtocolError("missing required 'value'", { instance: message });
                    return message;
                };

                /**
                 * Creates a DogmaAttributes message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {esf.TypeDogma.TypeDogmaEntry.DogmaAttributes} DogmaAttributes
                 */
                DogmaAttributes.fromObject = function fromObject(object) {
                    if (object instanceof $root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes)
                        return object;
                    let message = new $root.esf.TypeDogma.TypeDogmaEntry.DogmaAttributes();
                    if (object.attributeID != null)
                        message.attributeID = object.attributeID | 0;
                    if (object.value != null)
                        message.value = Number(object.value);
                    return message;
                };

                /**
                 * Creates a plain object from a DogmaAttributes message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @static
                 * @param {esf.TypeDogma.TypeDogmaEntry.DogmaAttributes} message DogmaAttributes
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                DogmaAttributes.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    let object = {};
                    if (options.defaults) {
                        object.attributeID = 0;
                        object.value = 0;
                    }
                    if (message.attributeID != null && message.hasOwnProperty("attributeID"))
                        object.attributeID = message.attributeID;
                    if (message.value != null && message.hasOwnProperty("value"))
                        object.value = options.json && !isFinite(message.value) ? String(message.value) : message.value;
                    return object;
                };

                /**
                 * Converts this DogmaAttributes to JSON.
                 * @function toJSON
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                DogmaAttributes.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                /**
                 * Gets the default type url for DogmaAttributes
                 * @function getTypeUrl
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaAttributes
                 * @static
                 * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns {string} The default type url
                 */
                DogmaAttributes.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                    if (typeUrlPrefix === undefined) {
                        typeUrlPrefix = "type.googleapis.com";
                    }
                    return typeUrlPrefix + "/esf.TypeDogma.TypeDogmaEntry.DogmaAttributes";
                };

                return DogmaAttributes;
            })();

            TypeDogmaEntry.DogmaEffects = (function() {

                /**
                 * Properties of a DogmaEffects.
                 * @memberof esf.TypeDogma.TypeDogmaEntry
                 * @interface IDogmaEffects
                 * @property {number} effectID DogmaEffects effectID
                 * @property {boolean} isDefault DogmaEffects isDefault
                 */

                /**
                 * Constructs a new DogmaEffects.
                 * @memberof esf.TypeDogma.TypeDogmaEntry
                 * @classdesc Represents a DogmaEffects.
                 * @implements IDogmaEffects
                 * @constructor
                 * @param {esf.TypeDogma.TypeDogmaEntry.IDogmaEffects=} [properties] Properties to set
                 */
                function DogmaEffects(properties) {
                    if (properties)
                        for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * DogmaEffects effectID.
                 * @member {number} effectID
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @instance
                 */
                DogmaEffects.prototype.effectID = 0;

                /**
                 * DogmaEffects isDefault.
                 * @member {boolean} isDefault
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @instance
                 */
                DogmaEffects.prototype.isDefault = false;

                /**
                 * Decodes a DogmaEffects message from the specified reader or buffer.
                 * @function decode
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {esf.TypeDogma.TypeDogmaEntry.DogmaEffects} DogmaEffects
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                DogmaEffects.decode = function decode(reader, length, error) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects();
                    while (reader.pos < end) {
                        let tag = reader.uint32();
                        if (tag === error)
                            break;
                        switch (tag >>> 3) {
                        case 1: {
                                message.effectID = reader.int32();
                                break;
                            }
                        case 2: {
                                message.isDefault = reader.bool();
                                break;
                            }
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    if (!message.hasOwnProperty("effectID"))
                        throw $util.ProtocolError("missing required 'effectID'", { instance: message });
                    if (!message.hasOwnProperty("isDefault"))
                        throw $util.ProtocolError("missing required 'isDefault'", { instance: message });
                    return message;
                };

                /**
                 * Creates a DogmaEffects message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {esf.TypeDogma.TypeDogmaEntry.DogmaEffects} DogmaEffects
                 */
                DogmaEffects.fromObject = function fromObject(object) {
                    if (object instanceof $root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects)
                        return object;
                    let message = new $root.esf.TypeDogma.TypeDogmaEntry.DogmaEffects();
                    if (object.effectID != null)
                        message.effectID = object.effectID | 0;
                    if (object.isDefault != null)
                        message.isDefault = Boolean(object.isDefault);
                    return message;
                };

                /**
                 * Creates a plain object from a DogmaEffects message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @static
                 * @param {esf.TypeDogma.TypeDogmaEntry.DogmaEffects} message DogmaEffects
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                DogmaEffects.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    let object = {};
                    if (options.defaults) {
                        object.effectID = 0;
                        object.isDefault = false;
                    }
                    if (message.effectID != null && message.hasOwnProperty("effectID"))
                        object.effectID = message.effectID;
                    if (message.isDefault != null && message.hasOwnProperty("isDefault"))
                        object.isDefault = message.isDefault;
                    return object;
                };

                /**
                 * Converts this DogmaEffects to JSON.
                 * @function toJSON
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                DogmaEffects.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                /**
                 * Gets the default type url for DogmaEffects
                 * @function getTypeUrl
                 * @memberof esf.TypeDogma.TypeDogmaEntry.DogmaEffects
                 * @static
                 * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns {string} The default type url
                 */
                DogmaEffects.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                    if (typeUrlPrefix === undefined) {
                        typeUrlPrefix = "type.googleapis.com";
                    }
                    return typeUrlPrefix + "/esf.TypeDogma.TypeDogmaEntry.DogmaEffects";
                };

                return DogmaEffects;
            })();

            return TypeDogmaEntry;
        })();

        return TypeDogma;
    })();

    esf.Types = (function() {

        /**
         * Properties of a Types.
         * @memberof esf
         * @interface ITypes
         * @property {Object.<string,esf.Types.IType>|null} [entries] Types entries
         */

        /**
         * Constructs a new Types.
         * @memberof esf
         * @classdesc Represents a Types.
         * @implements ITypes
         * @constructor
         * @param {esf.ITypes=} [properties] Properties to set
         */
        function Types(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Types entries.
         * @member {Object.<string,esf.Types.IType>} entries
         * @memberof esf.Types
         * @instance
         */
        Types.prototype.entries = $util.emptyObject;

        /**
         * Decodes a Types message from the specified reader or buffer.
         * @function decode
         * @memberof esf.Types
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.Types} Types
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Types.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Types(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.Types.Type.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a Types message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.Types
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.Types} Types
         */
        Types.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.Types)
                return object;
            let message = new $root.esf.Types();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.Types.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.Types.entries: object expected");
                    message.entries[keys[i]] = $root.esf.Types.Type.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a Types message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.Types
         * @static
         * @param {esf.Types} message Types
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Types.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.Types.Type.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this Types to JSON.
         * @function toJSON
         * @memberof esf.Types
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Types.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for Types
         * @function getTypeUrl
         * @memberof esf.Types
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        Types.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.Types";
        };

        Types.Type = (function() {

            /**
             * Properties of a Type.
             * @memberof esf.Types
             * @interface IType
             * @property {string} name Type name
             * @property {number} groupID Type groupID
             * @property {number} categoryID Type categoryID
             * @property {boolean} published Type published
             * @property {number|null} [factionID] Type factionID
             * @property {number|null} [marketGroupID] Type marketGroupID
             * @property {number|null} [metaGroupID] Type metaGroupID
             * @property {number|null} [capacity] Type capacity
             * @property {number|null} [mass] Type mass
             * @property {number|null} [radius] Type radius
             * @property {number|null} [volume] Type volume
             */

            /**
             * Constructs a new Type.
             * @memberof esf.Types
             * @classdesc Represents a Type.
             * @implements IType
             * @constructor
             * @param {esf.Types.IType=} [properties] Properties to set
             */
            function Type(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Type name.
             * @member {string} name
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.name = "";

            /**
             * Type groupID.
             * @member {number} groupID
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.groupID = 0;

            /**
             * Type categoryID.
             * @member {number} categoryID
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.categoryID = 0;

            /**
             * Type published.
             * @member {boolean} published
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.published = false;

            /**
             * Type factionID.
             * @member {number} factionID
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.factionID = 0;

            /**
             * Type marketGroupID.
             * @member {number} marketGroupID
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.marketGroupID = 0;

            /**
             * Type metaGroupID.
             * @member {number} metaGroupID
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.metaGroupID = 0;

            /**
             * Type capacity.
             * @member {number} capacity
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.capacity = 0;

            /**
             * Type mass.
             * @member {number} mass
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.mass = 0;

            /**
             * Type radius.
             * @member {number} radius
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.radius = 0;

            /**
             * Type volume.
             * @member {number} volume
             * @memberof esf.Types.Type
             * @instance
             */
            Type.prototype.volume = 0;

            /**
             * Decodes a Type message from the specified reader or buffer.
             * @function decode
             * @memberof esf.Types.Type
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.Types.Type} Type
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Type.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Types.Type();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.groupID = reader.int32();
                            break;
                        }
                    case 3: {
                            message.categoryID = reader.int32();
                            break;
                        }
                    case 4: {
                            message.published = reader.bool();
                            break;
                        }
                    case 5: {
                            message.factionID = reader.int32();
                            break;
                        }
                    case 6: {
                            message.marketGroupID = reader.int32();
                            break;
                        }
                    case 7: {
                            message.metaGroupID = reader.int32();
                            break;
                        }
                    case 8: {
                            message.capacity = reader.float();
                            break;
                        }
                    case 9: {
                            message.mass = reader.float();
                            break;
                        }
                    case 10: {
                            message.radius = reader.float();
                            break;
                        }
                    case 11: {
                            message.volume = reader.float();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                if (!message.hasOwnProperty("groupID"))
                    throw $util.ProtocolError("missing required 'groupID'", { instance: message });
                if (!message.hasOwnProperty("categoryID"))
                    throw $util.ProtocolError("missing required 'categoryID'", { instance: message });
                if (!message.hasOwnProperty("published"))
                    throw $util.ProtocolError("missing required 'published'", { instance: message });
                return message;
            };

            /**
             * Creates a Type message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.Types.Type
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.Types.Type} Type
             */
            Type.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.Types.Type)
                    return object;
                let message = new $root.esf.Types.Type();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.groupID != null)
                    message.groupID = object.groupID | 0;
                if (object.categoryID != null)
                    message.categoryID = object.categoryID | 0;
                if (object.published != null)
                    message.published = Boolean(object.published);
                if (object.factionID != null)
                    message.factionID = object.factionID | 0;
                if (object.marketGroupID != null)
                    message.marketGroupID = object.marketGroupID | 0;
                if (object.metaGroupID != null)
                    message.metaGroupID = object.metaGroupID | 0;
                if (object.capacity != null)
                    message.capacity = Number(object.capacity);
                if (object.mass != null)
                    message.mass = Number(object.mass);
                if (object.radius != null)
                    message.radius = Number(object.radius);
                if (object.volume != null)
                    message.volume = Number(object.volume);
                return message;
            };

            /**
             * Creates a plain object from a Type message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.Types.Type
             * @static
             * @param {esf.Types.Type} message Type
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Type.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.groupID = 0;
                    object.categoryID = 0;
                    object.published = false;
                    object.factionID = 0;
                    object.marketGroupID = 0;
                    object.metaGroupID = 0;
                    object.capacity = 0;
                    object.mass = 0;
                    object.radius = 0;
                    object.volume = 0;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.groupID != null && message.hasOwnProperty("groupID"))
                    object.groupID = message.groupID;
                if (message.categoryID != null && message.hasOwnProperty("categoryID"))
                    object.categoryID = message.categoryID;
                if (message.published != null && message.hasOwnProperty("published"))
                    object.published = message.published;
                if (message.factionID != null && message.hasOwnProperty("factionID"))
                    object.factionID = message.factionID;
                if (message.marketGroupID != null && message.hasOwnProperty("marketGroupID"))
                    object.marketGroupID = message.marketGroupID;
                if (message.metaGroupID != null && message.hasOwnProperty("metaGroupID"))
                    object.metaGroupID = message.metaGroupID;
                if (message.capacity != null && message.hasOwnProperty("capacity"))
                    object.capacity = options.json && !isFinite(message.capacity) ? String(message.capacity) : message.capacity;
                if (message.mass != null && message.hasOwnProperty("mass"))
                    object.mass = options.json && !isFinite(message.mass) ? String(message.mass) : message.mass;
                if (message.radius != null && message.hasOwnProperty("radius"))
                    object.radius = options.json && !isFinite(message.radius) ? String(message.radius) : message.radius;
                if (message.volume != null && message.hasOwnProperty("volume"))
                    object.volume = options.json && !isFinite(message.volume) ? String(message.volume) : message.volume;
                return object;
            };

            /**
             * Converts this Type to JSON.
             * @function toJSON
             * @memberof esf.Types.Type
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Type.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Type
             * @function getTypeUrl
             * @memberof esf.Types.Type
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Type.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.Types.Type";
            };

            return Type;
        })();

        return Types;
    })();

    esf.Categories = (function() {

        /**
         * Properties of a Categories.
         * @memberof esf
         * @interface ICategories
         * @property {Object.<string,esf.Categories.ICategory>|null} [entries] Categories entries
         */

        /**
         * Constructs a new Categories.
         * @memberof esf
         * @classdesc Represents a Categories.
         * @implements ICategories
         * @constructor
         * @param {esf.ICategories=} [properties] Properties to set
         */
        function Categories(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Categories entries.
         * @member {Object.<string,esf.Categories.ICategory>} entries
         * @memberof esf.Categories
         * @instance
         */
        Categories.prototype.entries = $util.emptyObject;

        /**
         * Decodes a Categories message from the specified reader or buffer.
         * @function decode
         * @memberof esf.Categories
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.Categories} Categories
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Categories.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Categories(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.Categories.Category.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a Categories message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.Categories
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.Categories} Categories
         */
        Categories.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.Categories)
                return object;
            let message = new $root.esf.Categories();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.Categories.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.Categories.entries: object expected");
                    message.entries[keys[i]] = $root.esf.Categories.Category.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a Categories message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.Categories
         * @static
         * @param {esf.Categories} message Categories
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Categories.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.Categories.Category.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this Categories to JSON.
         * @function toJSON
         * @memberof esf.Categories
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Categories.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for Categories
         * @function getTypeUrl
         * @memberof esf.Categories
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        Categories.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.Categories";
        };

        Categories.Category = (function() {

            /**
             * Properties of a Category.
             * @memberof esf.Categories
             * @interface ICategory
             * @property {string} name Category name
             * @property {boolean} published Category published
             */

            /**
             * Constructs a new Category.
             * @memberof esf.Categories
             * @classdesc Represents a Category.
             * @implements ICategory
             * @constructor
             * @param {esf.Categories.ICategory=} [properties] Properties to set
             */
            function Category(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Category name.
             * @member {string} name
             * @memberof esf.Categories.Category
             * @instance
             */
            Category.prototype.name = "";

            /**
             * Category published.
             * @member {boolean} published
             * @memberof esf.Categories.Category
             * @instance
             */
            Category.prototype.published = false;

            /**
             * Decodes a Category message from the specified reader or buffer.
             * @function decode
             * @memberof esf.Categories.Category
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.Categories.Category} Category
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Category.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Categories.Category();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.published = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                if (!message.hasOwnProperty("published"))
                    throw $util.ProtocolError("missing required 'published'", { instance: message });
                return message;
            };

            /**
             * Creates a Category message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.Categories.Category
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.Categories.Category} Category
             */
            Category.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.Categories.Category)
                    return object;
                let message = new $root.esf.Categories.Category();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.published != null)
                    message.published = Boolean(object.published);
                return message;
            };

            /**
             * Creates a plain object from a Category message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.Categories.Category
             * @static
             * @param {esf.Categories.Category} message Category
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Category.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.published = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.published != null && message.hasOwnProperty("published"))
                    object.published = message.published;
                return object;
            };

            /**
             * Converts this Category to JSON.
             * @function toJSON
             * @memberof esf.Categories.Category
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Category.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Category
             * @function getTypeUrl
             * @memberof esf.Categories.Category
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Category.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.Categories.Category";
            };

            return Category;
        })();

        return Categories;
    })();

    esf.Groups = (function() {

        /**
         * Properties of a Groups.
         * @memberof esf
         * @interface IGroups
         * @property {Object.<string,esf.Groups.IGroup>|null} [entries] Groups entries
         */

        /**
         * Constructs a new Groups.
         * @memberof esf
         * @classdesc Represents a Groups.
         * @implements IGroups
         * @constructor
         * @param {esf.IGroups=} [properties] Properties to set
         */
        function Groups(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * Groups entries.
         * @member {Object.<string,esf.Groups.IGroup>} entries
         * @memberof esf.Groups
         * @instance
         */
        Groups.prototype.entries = $util.emptyObject;

        /**
         * Decodes a Groups message from the specified reader or buffer.
         * @function decode
         * @memberof esf.Groups
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.Groups} Groups
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        Groups.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Groups(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.Groups.Group.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a Groups message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.Groups
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.Groups} Groups
         */
        Groups.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.Groups)
                return object;
            let message = new $root.esf.Groups();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.Groups.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.Groups.entries: object expected");
                    message.entries[keys[i]] = $root.esf.Groups.Group.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a Groups message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.Groups
         * @static
         * @param {esf.Groups} message Groups
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        Groups.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.Groups.Group.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this Groups to JSON.
         * @function toJSON
         * @memberof esf.Groups
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        Groups.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for Groups
         * @function getTypeUrl
         * @memberof esf.Groups
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        Groups.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.Groups";
        };

        Groups.Group = (function() {

            /**
             * Properties of a Group.
             * @memberof esf.Groups
             * @interface IGroup
             * @property {string} name Group name
             * @property {number} categoryID Group categoryID
             * @property {boolean} published Group published
             */

            /**
             * Constructs a new Group.
             * @memberof esf.Groups
             * @classdesc Represents a Group.
             * @implements IGroup
             * @constructor
             * @param {esf.Groups.IGroup=} [properties] Properties to set
             */
            function Group(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Group name.
             * @member {string} name
             * @memberof esf.Groups.Group
             * @instance
             */
            Group.prototype.name = "";

            /**
             * Group categoryID.
             * @member {number} categoryID
             * @memberof esf.Groups.Group
             * @instance
             */
            Group.prototype.categoryID = 0;

            /**
             * Group published.
             * @member {boolean} published
             * @memberof esf.Groups.Group
             * @instance
             */
            Group.prototype.published = false;

            /**
             * Decodes a Group message from the specified reader or buffer.
             * @function decode
             * @memberof esf.Groups.Group
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.Groups.Group} Group
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Group.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.Groups.Group();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.categoryID = reader.int32();
                            break;
                        }
                    case 3: {
                            message.published = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                if (!message.hasOwnProperty("categoryID"))
                    throw $util.ProtocolError("missing required 'categoryID'", { instance: message });
                if (!message.hasOwnProperty("published"))
                    throw $util.ProtocolError("missing required 'published'", { instance: message });
                return message;
            };

            /**
             * Creates a Group message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.Groups.Group
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.Groups.Group} Group
             */
            Group.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.Groups.Group)
                    return object;
                let message = new $root.esf.Groups.Group();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.categoryID != null)
                    message.categoryID = object.categoryID | 0;
                if (object.published != null)
                    message.published = Boolean(object.published);
                return message;
            };

            /**
             * Creates a plain object from a Group message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.Groups.Group
             * @static
             * @param {esf.Groups.Group} message Group
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Group.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.categoryID = 0;
                    object.published = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.categoryID != null && message.hasOwnProperty("categoryID"))
                    object.categoryID = message.categoryID;
                if (message.published != null && message.hasOwnProperty("published"))
                    object.published = message.published;
                return object;
            };

            /**
             * Converts this Group to JSON.
             * @function toJSON
             * @memberof esf.Groups.Group
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Group.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Group
             * @function getTypeUrl
             * @memberof esf.Groups.Group
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Group.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.Groups.Group";
            };

            return Group;
        })();

        return Groups;
    })();

    esf.MarketGroups = (function() {

        /**
         * Properties of a MarketGroups.
         * @memberof esf
         * @interface IMarketGroups
         * @property {Object.<string,esf.MarketGroups.IMarketGroup>|null} [entries] MarketGroups entries
         */

        /**
         * Constructs a new MarketGroups.
         * @memberof esf
         * @classdesc Represents a MarketGroups.
         * @implements IMarketGroups
         * @constructor
         * @param {esf.IMarketGroups=} [properties] Properties to set
         */
        function MarketGroups(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * MarketGroups entries.
         * @member {Object.<string,esf.MarketGroups.IMarketGroup>} entries
         * @memberof esf.MarketGroups
         * @instance
         */
        MarketGroups.prototype.entries = $util.emptyObject;

        /**
         * Decodes a MarketGroups message from the specified reader or buffer.
         * @function decode
         * @memberof esf.MarketGroups
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.MarketGroups} MarketGroups
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        MarketGroups.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.MarketGroups(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.MarketGroups.MarketGroup.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a MarketGroups message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.MarketGroups
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.MarketGroups} MarketGroups
         */
        MarketGroups.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.MarketGroups)
                return object;
            let message = new $root.esf.MarketGroups();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.MarketGroups.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.MarketGroups.entries: object expected");
                    message.entries[keys[i]] = $root.esf.MarketGroups.MarketGroup.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a MarketGroups message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.MarketGroups
         * @static
         * @param {esf.MarketGroups} message MarketGroups
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        MarketGroups.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.MarketGroups.MarketGroup.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this MarketGroups to JSON.
         * @function toJSON
         * @memberof esf.MarketGroups
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        MarketGroups.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for MarketGroups
         * @function getTypeUrl
         * @memberof esf.MarketGroups
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        MarketGroups.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.MarketGroups";
        };

        MarketGroups.MarketGroup = (function() {

            /**
             * Properties of a MarketGroup.
             * @memberof esf.MarketGroups
             * @interface IMarketGroup
             * @property {string} name MarketGroup name
             * @property {number|null} [parentGroupID] MarketGroup parentGroupID
             * @property {number|null} [iconID] MarketGroup iconID
             */

            /**
             * Constructs a new MarketGroup.
             * @memberof esf.MarketGroups
             * @classdesc Represents a MarketGroup.
             * @implements IMarketGroup
             * @constructor
             * @param {esf.MarketGroups.IMarketGroup=} [properties] Properties to set
             */
            function MarketGroup(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * MarketGroup name.
             * @member {string} name
             * @memberof esf.MarketGroups.MarketGroup
             * @instance
             */
            MarketGroup.prototype.name = "";

            /**
             * MarketGroup parentGroupID.
             * @member {number} parentGroupID
             * @memberof esf.MarketGroups.MarketGroup
             * @instance
             */
            MarketGroup.prototype.parentGroupID = 0;

            /**
             * MarketGroup iconID.
             * @member {number} iconID
             * @memberof esf.MarketGroups.MarketGroup
             * @instance
             */
            MarketGroup.prototype.iconID = 0;

            /**
             * Decodes a MarketGroup message from the specified reader or buffer.
             * @function decode
             * @memberof esf.MarketGroups.MarketGroup
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.MarketGroups.MarketGroup} MarketGroup
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            MarketGroup.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.MarketGroups.MarketGroup();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.parentGroupID = reader.int32();
                            break;
                        }
                    case 3: {
                            message.iconID = reader.int32();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                return message;
            };

            /**
             * Creates a MarketGroup message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.MarketGroups.MarketGroup
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.MarketGroups.MarketGroup} MarketGroup
             */
            MarketGroup.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.MarketGroups.MarketGroup)
                    return object;
                let message = new $root.esf.MarketGroups.MarketGroup();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.parentGroupID != null)
                    message.parentGroupID = object.parentGroupID | 0;
                if (object.iconID != null)
                    message.iconID = object.iconID | 0;
                return message;
            };

            /**
             * Creates a plain object from a MarketGroup message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.MarketGroups.MarketGroup
             * @static
             * @param {esf.MarketGroups.MarketGroup} message MarketGroup
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            MarketGroup.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.parentGroupID = 0;
                    object.iconID = 0;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.parentGroupID != null && message.hasOwnProperty("parentGroupID"))
                    object.parentGroupID = message.parentGroupID;
                if (message.iconID != null && message.hasOwnProperty("iconID"))
                    object.iconID = message.iconID;
                return object;
            };

            /**
             * Converts this MarketGroup to JSON.
             * @function toJSON
             * @memberof esf.MarketGroups.MarketGroup
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            MarketGroup.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for MarketGroup
             * @function getTypeUrl
             * @memberof esf.MarketGroups.MarketGroup
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            MarketGroup.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.MarketGroups.MarketGroup";
            };

            return MarketGroup;
        })();

        return MarketGroups;
    })();

    esf.DogmaAttributes = (function() {

        /**
         * Properties of a DogmaAttributes.
         * @memberof esf
         * @interface IDogmaAttributes
         * @property {Object.<string,esf.DogmaAttributes.IDogmaAttribute>|null} [entries] DogmaAttributes entries
         */

        /**
         * Constructs a new DogmaAttributes.
         * @memberof esf
         * @classdesc Represents a DogmaAttributes.
         * @implements IDogmaAttributes
         * @constructor
         * @param {esf.IDogmaAttributes=} [properties] Properties to set
         */
        function DogmaAttributes(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * DogmaAttributes entries.
         * @member {Object.<string,esf.DogmaAttributes.IDogmaAttribute>} entries
         * @memberof esf.DogmaAttributes
         * @instance
         */
        DogmaAttributes.prototype.entries = $util.emptyObject;

        /**
         * Decodes a DogmaAttributes message from the specified reader or buffer.
         * @function decode
         * @memberof esf.DogmaAttributes
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.DogmaAttributes} DogmaAttributes
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DogmaAttributes.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.DogmaAttributes(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.DogmaAttributes.DogmaAttribute.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a DogmaAttributes message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.DogmaAttributes
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.DogmaAttributes} DogmaAttributes
         */
        DogmaAttributes.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.DogmaAttributes)
                return object;
            let message = new $root.esf.DogmaAttributes();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.DogmaAttributes.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.DogmaAttributes.entries: object expected");
                    message.entries[keys[i]] = $root.esf.DogmaAttributes.DogmaAttribute.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a DogmaAttributes message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.DogmaAttributes
         * @static
         * @param {esf.DogmaAttributes} message DogmaAttributes
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        DogmaAttributes.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.DogmaAttributes.DogmaAttribute.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this DogmaAttributes to JSON.
         * @function toJSON
         * @memberof esf.DogmaAttributes
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        DogmaAttributes.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for DogmaAttributes
         * @function getTypeUrl
         * @memberof esf.DogmaAttributes
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        DogmaAttributes.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.DogmaAttributes";
        };

        DogmaAttributes.DogmaAttribute = (function() {

            /**
             * Properties of a DogmaAttribute.
             * @memberof esf.DogmaAttributes
             * @interface IDogmaAttribute
             * @property {string} name DogmaAttribute name
             * @property {boolean} published DogmaAttribute published
             * @property {number} defaultValue DogmaAttribute defaultValue
             * @property {boolean} highIsGood DogmaAttribute highIsGood
             * @property {boolean} stackable DogmaAttribute stackable
             */

            /**
             * Constructs a new DogmaAttribute.
             * @memberof esf.DogmaAttributes
             * @classdesc Represents a DogmaAttribute.
             * @implements IDogmaAttribute
             * @constructor
             * @param {esf.DogmaAttributes.IDogmaAttribute=} [properties] Properties to set
             */
            function DogmaAttribute(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * DogmaAttribute name.
             * @member {string} name
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             */
            DogmaAttribute.prototype.name = "";

            /**
             * DogmaAttribute published.
             * @member {boolean} published
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             */
            DogmaAttribute.prototype.published = false;

            /**
             * DogmaAttribute defaultValue.
             * @member {number} defaultValue
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             */
            DogmaAttribute.prototype.defaultValue = 0;

            /**
             * DogmaAttribute highIsGood.
             * @member {boolean} highIsGood
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             */
            DogmaAttribute.prototype.highIsGood = false;

            /**
             * DogmaAttribute stackable.
             * @member {boolean} stackable
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             */
            DogmaAttribute.prototype.stackable = false;

            /**
             * Decodes a DogmaAttribute message from the specified reader or buffer.
             * @function decode
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.DogmaAttributes.DogmaAttribute} DogmaAttribute
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DogmaAttribute.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.DogmaAttributes.DogmaAttribute();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.published = reader.bool();
                            break;
                        }
                    case 3: {
                            message.defaultValue = reader.float();
                            break;
                        }
                    case 4: {
                            message.highIsGood = reader.bool();
                            break;
                        }
                    case 5: {
                            message.stackable = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                if (!message.hasOwnProperty("published"))
                    throw $util.ProtocolError("missing required 'published'", { instance: message });
                if (!message.hasOwnProperty("defaultValue"))
                    throw $util.ProtocolError("missing required 'defaultValue'", { instance: message });
                if (!message.hasOwnProperty("highIsGood"))
                    throw $util.ProtocolError("missing required 'highIsGood'", { instance: message });
                if (!message.hasOwnProperty("stackable"))
                    throw $util.ProtocolError("missing required 'stackable'", { instance: message });
                return message;
            };

            /**
             * Creates a DogmaAttribute message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.DogmaAttributes.DogmaAttribute} DogmaAttribute
             */
            DogmaAttribute.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.DogmaAttributes.DogmaAttribute)
                    return object;
                let message = new $root.esf.DogmaAttributes.DogmaAttribute();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.published != null)
                    message.published = Boolean(object.published);
                if (object.defaultValue != null)
                    message.defaultValue = Number(object.defaultValue);
                if (object.highIsGood != null)
                    message.highIsGood = Boolean(object.highIsGood);
                if (object.stackable != null)
                    message.stackable = Boolean(object.stackable);
                return message;
            };

            /**
             * Creates a plain object from a DogmaAttribute message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @static
             * @param {esf.DogmaAttributes.DogmaAttribute} message DogmaAttribute
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            DogmaAttribute.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.published = false;
                    object.defaultValue = 0;
                    object.highIsGood = false;
                    object.stackable = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.published != null && message.hasOwnProperty("published"))
                    object.published = message.published;
                if (message.defaultValue != null && message.hasOwnProperty("defaultValue"))
                    object.defaultValue = options.json && !isFinite(message.defaultValue) ? String(message.defaultValue) : message.defaultValue;
                if (message.highIsGood != null && message.hasOwnProperty("highIsGood"))
                    object.highIsGood = message.highIsGood;
                if (message.stackable != null && message.hasOwnProperty("stackable"))
                    object.stackable = message.stackable;
                return object;
            };

            /**
             * Converts this DogmaAttribute to JSON.
             * @function toJSON
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            DogmaAttribute.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for DogmaAttribute
             * @function getTypeUrl
             * @memberof esf.DogmaAttributes.DogmaAttribute
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            DogmaAttribute.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.DogmaAttributes.DogmaAttribute";
            };

            return DogmaAttribute;
        })();

        return DogmaAttributes;
    })();

    esf.DogmaEffects = (function() {

        /**
         * Properties of a DogmaEffects.
         * @memberof esf
         * @interface IDogmaEffects
         * @property {Object.<string,esf.DogmaEffects.IDogmaEffect>|null} [entries] DogmaEffects entries
         */

        /**
         * Constructs a new DogmaEffects.
         * @memberof esf
         * @classdesc Represents a DogmaEffects.
         * @implements IDogmaEffects
         * @constructor
         * @param {esf.IDogmaEffects=} [properties] Properties to set
         */
        function DogmaEffects(properties) {
            this.entries = {};
            if (properties)
                for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                    if (properties[keys[i]] != null)
                        this[keys[i]] = properties[keys[i]];
        }

        /**
         * DogmaEffects entries.
         * @member {Object.<string,esf.DogmaEffects.IDogmaEffect>} entries
         * @memberof esf.DogmaEffects
         * @instance
         */
        DogmaEffects.prototype.entries = $util.emptyObject;

        /**
         * Decodes a DogmaEffects message from the specified reader or buffer.
         * @function decode
         * @memberof esf.DogmaEffects
         * @static
         * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
         * @param {number} [length] Message length if known beforehand
         * @returns {esf.DogmaEffects} DogmaEffects
         * @throws {Error} If the payload is not a reader or valid buffer
         * @throws {$protobuf.util.ProtocolError} If required fields are missing
         */
        DogmaEffects.decode = function decode(reader, length, error) {
            if (!(reader instanceof $Reader))
                reader = $Reader.create(reader);
            let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.DogmaEffects(), key, value;
            while (reader.pos < end) {
                let tag = reader.uint32();
                if (tag === error)
                    break;
                switch (tag >>> 3) {
                case 1: {
                        if (message.entries === $util.emptyObject)
                            message.entries = {};
                        let end2 = reader.uint32() + reader.pos;
                        key = 0;
                        value = null;
                        while (reader.pos < end2) {
                            let tag2 = reader.uint32();
                            switch (tag2 >>> 3) {
                            case 1:
                                key = reader.int32();
                                break;
                            case 2:
                                value = $root.esf.DogmaEffects.DogmaEffect.decode(reader, reader.uint32());
                                break;
                            default:
                                reader.skipType(tag2 & 7);
                                break;
                            }
                        }
                        message.entries[key] = value;
                        break;
                    }
                default:
                    reader.skipType(tag & 7);
                    break;
                }
            }
            return message;
        };

        /**
         * Creates a DogmaEffects message from a plain object. Also converts values to their respective internal types.
         * @function fromObject
         * @memberof esf.DogmaEffects
         * @static
         * @param {Object.<string,*>} object Plain object
         * @returns {esf.DogmaEffects} DogmaEffects
         */
        DogmaEffects.fromObject = function fromObject(object) {
            if (object instanceof $root.esf.DogmaEffects)
                return object;
            let message = new $root.esf.DogmaEffects();
            if (object.entries) {
                if (typeof object.entries !== "object")
                    throw TypeError(".esf.DogmaEffects.entries: object expected");
                message.entries = {};
                for (let keys = Object.keys(object.entries), i = 0; i < keys.length; ++i) {
                    if (typeof object.entries[keys[i]] !== "object")
                        throw TypeError(".esf.DogmaEffects.entries: object expected");
                    message.entries[keys[i]] = $root.esf.DogmaEffects.DogmaEffect.fromObject(object.entries[keys[i]]);
                }
            }
            return message;
        };

        /**
         * Creates a plain object from a DogmaEffects message. Also converts values to other types if specified.
         * @function toObject
         * @memberof esf.DogmaEffects
         * @static
         * @param {esf.DogmaEffects} message DogmaEffects
         * @param {$protobuf.IConversionOptions} [options] Conversion options
         * @returns {Object.<string,*>} Plain object
         */
        DogmaEffects.toObject = function toObject(message, options) {
            if (!options)
                options = {};
            let object = {};
            if (options.objects || options.defaults)
                object.entries = {};
            let keys2;
            if (message.entries && (keys2 = Object.keys(message.entries)).length) {
                object.entries = {};
                for (let j = 0; j < keys2.length; ++j)
                    object.entries[keys2[j]] = $root.esf.DogmaEffects.DogmaEffect.toObject(message.entries[keys2[j]], options);
            }
            return object;
        };

        /**
         * Converts this DogmaEffects to JSON.
         * @function toJSON
         * @memberof esf.DogmaEffects
         * @instance
         * @returns {Object.<string,*>} JSON object
         */
        DogmaEffects.prototype.toJSON = function toJSON() {
            return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
        };

        /**
         * Gets the default type url for DogmaEffects
         * @function getTypeUrl
         * @memberof esf.DogmaEffects
         * @static
         * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
         * @returns {string} The default type url
         */
        DogmaEffects.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
            if (typeUrlPrefix === undefined) {
                typeUrlPrefix = "type.googleapis.com";
            }
            return typeUrlPrefix + "/esf.DogmaEffects";
        };

        DogmaEffects.DogmaEffect = (function() {

            /**
             * Properties of a DogmaEffect.
             * @memberof esf.DogmaEffects
             * @interface IDogmaEffect
             * @property {string} name DogmaEffect name
             * @property {number} effectCategory DogmaEffect effectCategory
             * @property {boolean} electronicChance DogmaEffect electronicChance
             * @property {boolean} isAssistance DogmaEffect isAssistance
             * @property {boolean} isOffensive DogmaEffect isOffensive
             * @property {boolean} isWarpSafe DogmaEffect isWarpSafe
             * @property {boolean} propulsionChance DogmaEffect propulsionChance
             * @property {boolean} rangeChance DogmaEffect rangeChance
             * @property {number|null} [dischargeAttributeID] DogmaEffect dischargeAttributeID
             * @property {number|null} [durationAttributeID] DogmaEffect durationAttributeID
             * @property {number|null} [rangeAttributeID] DogmaEffect rangeAttributeID
             * @property {number|null} [falloffAttributeID] DogmaEffect falloffAttributeID
             * @property {number|null} [trackingSpeedAttributeID] DogmaEffect trackingSpeedAttributeID
             * @property {number|null} [fittingUsageChanceAttributeID] DogmaEffect fittingUsageChanceAttributeID
             * @property {number|null} [resistanceAttributeID] DogmaEffect resistanceAttributeID
             * @property {Array.<esf.DogmaEffects.DogmaEffect.IModifierInfo>|null} [modifierInfo] DogmaEffect modifierInfo
             */

            /**
             * Constructs a new DogmaEffect.
             * @memberof esf.DogmaEffects
             * @classdesc Represents a DogmaEffect.
             * @implements IDogmaEffect
             * @constructor
             * @param {esf.DogmaEffects.IDogmaEffect=} [properties] Properties to set
             */
            function DogmaEffect(properties) {
                this.modifierInfo = [];
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * DogmaEffect name.
             * @member {string} name
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.name = "";

            /**
             * DogmaEffect effectCategory.
             * @member {number} effectCategory
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.effectCategory = 0;

            /**
             * DogmaEffect electronicChance.
             * @member {boolean} electronicChance
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.electronicChance = false;

            /**
             * DogmaEffect isAssistance.
             * @member {boolean} isAssistance
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.isAssistance = false;

            /**
             * DogmaEffect isOffensive.
             * @member {boolean} isOffensive
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.isOffensive = false;

            /**
             * DogmaEffect isWarpSafe.
             * @member {boolean} isWarpSafe
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.isWarpSafe = false;

            /**
             * DogmaEffect propulsionChance.
             * @member {boolean} propulsionChance
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.propulsionChance = false;

            /**
             * DogmaEffect rangeChance.
             * @member {boolean} rangeChance
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.rangeChance = false;

            /**
             * DogmaEffect dischargeAttributeID.
             * @member {number} dischargeAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.dischargeAttributeID = 0;

            /**
             * DogmaEffect durationAttributeID.
             * @member {number} durationAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.durationAttributeID = 0;

            /**
             * DogmaEffect rangeAttributeID.
             * @member {number} rangeAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.rangeAttributeID = 0;

            /**
             * DogmaEffect falloffAttributeID.
             * @member {number} falloffAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.falloffAttributeID = 0;

            /**
             * DogmaEffect trackingSpeedAttributeID.
             * @member {number} trackingSpeedAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.trackingSpeedAttributeID = 0;

            /**
             * DogmaEffect fittingUsageChanceAttributeID.
             * @member {number} fittingUsageChanceAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.fittingUsageChanceAttributeID = 0;

            /**
             * DogmaEffect resistanceAttributeID.
             * @member {number} resistanceAttributeID
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.resistanceAttributeID = 0;

            /**
             * DogmaEffect modifierInfo.
             * @member {Array.<esf.DogmaEffects.DogmaEffect.IModifierInfo>} modifierInfo
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             */
            DogmaEffect.prototype.modifierInfo = $util.emptyArray;

            /**
             * Decodes a DogmaEffect message from the specified reader or buffer.
             * @function decode
             * @memberof esf.DogmaEffects.DogmaEffect
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {esf.DogmaEffects.DogmaEffect} DogmaEffect
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            DogmaEffect.decode = function decode(reader, length, error) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.DogmaEffects.DogmaEffect();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    if (tag === error)
                        break;
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.effectCategory = reader.int32();
                            break;
                        }
                    case 3: {
                            message.electronicChance = reader.bool();
                            break;
                        }
                    case 4: {
                            message.isAssistance = reader.bool();
                            break;
                        }
                    case 5: {
                            message.isOffensive = reader.bool();
                            break;
                        }
                    case 6: {
                            message.isWarpSafe = reader.bool();
                            break;
                        }
                    case 7: {
                            message.propulsionChance = reader.bool();
                            break;
                        }
                    case 8: {
                            message.rangeChance = reader.bool();
                            break;
                        }
                    case 9: {
                            message.dischargeAttributeID = reader.int32();
                            break;
                        }
                    case 10: {
                            message.durationAttributeID = reader.int32();
                            break;
                        }
                    case 11: {
                            message.rangeAttributeID = reader.int32();
                            break;
                        }
                    case 12: {
                            message.falloffAttributeID = reader.int32();
                            break;
                        }
                    case 13: {
                            message.trackingSpeedAttributeID = reader.int32();
                            break;
                        }
                    case 14: {
                            message.fittingUsageChanceAttributeID = reader.int32();
                            break;
                        }
                    case 15: {
                            message.resistanceAttributeID = reader.int32();
                            break;
                        }
                    case 16: {
                            if (!(message.modifierInfo && message.modifierInfo.length))
                                message.modifierInfo = [];
                            message.modifierInfo.push($root.esf.DogmaEffects.DogmaEffect.ModifierInfo.decode(reader, reader.uint32()));
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                if (!message.hasOwnProperty("name"))
                    throw $util.ProtocolError("missing required 'name'", { instance: message });
                if (!message.hasOwnProperty("effectCategory"))
                    throw $util.ProtocolError("missing required 'effectCategory'", { instance: message });
                if (!message.hasOwnProperty("electronicChance"))
                    throw $util.ProtocolError("missing required 'electronicChance'", { instance: message });
                if (!message.hasOwnProperty("isAssistance"))
                    throw $util.ProtocolError("missing required 'isAssistance'", { instance: message });
                if (!message.hasOwnProperty("isOffensive"))
                    throw $util.ProtocolError("missing required 'isOffensive'", { instance: message });
                if (!message.hasOwnProperty("isWarpSafe"))
                    throw $util.ProtocolError("missing required 'isWarpSafe'", { instance: message });
                if (!message.hasOwnProperty("propulsionChance"))
                    throw $util.ProtocolError("missing required 'propulsionChance'", { instance: message });
                if (!message.hasOwnProperty("rangeChance"))
                    throw $util.ProtocolError("missing required 'rangeChance'", { instance: message });
                return message;
            };

            /**
             * Creates a DogmaEffect message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof esf.DogmaEffects.DogmaEffect
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {esf.DogmaEffects.DogmaEffect} DogmaEffect
             */
            DogmaEffect.fromObject = function fromObject(object) {
                if (object instanceof $root.esf.DogmaEffects.DogmaEffect)
                    return object;
                let message = new $root.esf.DogmaEffects.DogmaEffect();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.effectCategory != null)
                    message.effectCategory = object.effectCategory | 0;
                if (object.electronicChance != null)
                    message.electronicChance = Boolean(object.electronicChance);
                if (object.isAssistance != null)
                    message.isAssistance = Boolean(object.isAssistance);
                if (object.isOffensive != null)
                    message.isOffensive = Boolean(object.isOffensive);
                if (object.isWarpSafe != null)
                    message.isWarpSafe = Boolean(object.isWarpSafe);
                if (object.propulsionChance != null)
                    message.propulsionChance = Boolean(object.propulsionChance);
                if (object.rangeChance != null)
                    message.rangeChance = Boolean(object.rangeChance);
                if (object.dischargeAttributeID != null)
                    message.dischargeAttributeID = object.dischargeAttributeID | 0;
                if (object.durationAttributeID != null)
                    message.durationAttributeID = object.durationAttributeID | 0;
                if (object.rangeAttributeID != null)
                    message.rangeAttributeID = object.rangeAttributeID | 0;
                if (object.falloffAttributeID != null)
                    message.falloffAttributeID = object.falloffAttributeID | 0;
                if (object.trackingSpeedAttributeID != null)
                    message.trackingSpeedAttributeID = object.trackingSpeedAttributeID | 0;
                if (object.fittingUsageChanceAttributeID != null)
                    message.fittingUsageChanceAttributeID = object.fittingUsageChanceAttributeID | 0;
                if (object.resistanceAttributeID != null)
                    message.resistanceAttributeID = object.resistanceAttributeID | 0;
                if (object.modifierInfo) {
                    if (!Array.isArray(object.modifierInfo))
                        throw TypeError(".esf.DogmaEffects.DogmaEffect.modifierInfo: array expected");
                    message.modifierInfo = [];
                    for (let i = 0; i < object.modifierInfo.length; ++i) {
                        if (typeof object.modifierInfo[i] !== "object")
                            throw TypeError(".esf.DogmaEffects.DogmaEffect.modifierInfo: object expected");
                        message.modifierInfo[i] = $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.fromObject(object.modifierInfo[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a DogmaEffect message. Also converts values to other types if specified.
             * @function toObject
             * @memberof esf.DogmaEffects.DogmaEffect
             * @static
             * @param {esf.DogmaEffects.DogmaEffect} message DogmaEffect
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            DogmaEffect.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.arrays || options.defaults)
                    object.modifierInfo = [];
                if (options.defaults) {
                    object.name = "";
                    object.effectCategory = 0;
                    object.electronicChance = false;
                    object.isAssistance = false;
                    object.isOffensive = false;
                    object.isWarpSafe = false;
                    object.propulsionChance = false;
                    object.rangeChance = false;
                    object.dischargeAttributeID = 0;
                    object.durationAttributeID = 0;
                    object.rangeAttributeID = 0;
                    object.falloffAttributeID = 0;
                    object.trackingSpeedAttributeID = 0;
                    object.fittingUsageChanceAttributeID = 0;
                    object.resistanceAttributeID = 0;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.effectCategory != null && message.hasOwnProperty("effectCategory"))
                    object.effectCategory = message.effectCategory;
                if (message.electronicChance != null && message.hasOwnProperty("electronicChance"))
                    object.electronicChance = message.electronicChance;
                if (message.isAssistance != null && message.hasOwnProperty("isAssistance"))
                    object.isAssistance = message.isAssistance;
                if (message.isOffensive != null && message.hasOwnProperty("isOffensive"))
                    object.isOffensive = message.isOffensive;
                if (message.isWarpSafe != null && message.hasOwnProperty("isWarpSafe"))
                    object.isWarpSafe = message.isWarpSafe;
                if (message.propulsionChance != null && message.hasOwnProperty("propulsionChance"))
                    object.propulsionChance = message.propulsionChance;
                if (message.rangeChance != null && message.hasOwnProperty("rangeChance"))
                    object.rangeChance = message.rangeChance;
                if (message.dischargeAttributeID != null && message.hasOwnProperty("dischargeAttributeID"))
                    object.dischargeAttributeID = message.dischargeAttributeID;
                if (message.durationAttributeID != null && message.hasOwnProperty("durationAttributeID"))
                    object.durationAttributeID = message.durationAttributeID;
                if (message.rangeAttributeID != null && message.hasOwnProperty("rangeAttributeID"))
                    object.rangeAttributeID = message.rangeAttributeID;
                if (message.falloffAttributeID != null && message.hasOwnProperty("falloffAttributeID"))
                    object.falloffAttributeID = message.falloffAttributeID;
                if (message.trackingSpeedAttributeID != null && message.hasOwnProperty("trackingSpeedAttributeID"))
                    object.trackingSpeedAttributeID = message.trackingSpeedAttributeID;
                if (message.fittingUsageChanceAttributeID != null && message.hasOwnProperty("fittingUsageChanceAttributeID"))
                    object.fittingUsageChanceAttributeID = message.fittingUsageChanceAttributeID;
                if (message.resistanceAttributeID != null && message.hasOwnProperty("resistanceAttributeID"))
                    object.resistanceAttributeID = message.resistanceAttributeID;
                if (message.modifierInfo && message.modifierInfo.length) {
                    object.modifierInfo = [];
                    for (let j = 0; j < message.modifierInfo.length; ++j)
                        object.modifierInfo[j] = $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.toObject(message.modifierInfo[j], options);
                }
                return object;
            };

            /**
             * Converts this DogmaEffect to JSON.
             * @function toJSON
             * @memberof esf.DogmaEffects.DogmaEffect
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            DogmaEffect.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for DogmaEffect
             * @function getTypeUrl
             * @memberof esf.DogmaEffects.DogmaEffect
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            DogmaEffect.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/esf.DogmaEffects.DogmaEffect";
            };

            DogmaEffect.ModifierInfo = (function() {

                /**
                 * Properties of a ModifierInfo.
                 * @memberof esf.DogmaEffects.DogmaEffect
                 * @interface IModifierInfo
                 * @property {esf.DogmaEffects.DogmaEffect.ModifierInfo.Domain} domain ModifierInfo domain
                 * @property {esf.DogmaEffects.DogmaEffect.ModifierInfo.Func} func ModifierInfo func
                 * @property {number|null} [modifiedAttributeID] ModifierInfo modifiedAttributeID
                 * @property {number|null} [modifyingAttributeID] ModifierInfo modifyingAttributeID
                 * @property {number|null} [operation] ModifierInfo operation
                 * @property {number|null} [groupID] ModifierInfo groupID
                 * @property {number|null} [skillTypeID] ModifierInfo skillTypeID
                 */

                /**
                 * Constructs a new ModifierInfo.
                 * @memberof esf.DogmaEffects.DogmaEffect
                 * @classdesc Represents a ModifierInfo.
                 * @implements IModifierInfo
                 * @constructor
                 * @param {esf.DogmaEffects.DogmaEffect.IModifierInfo=} [properties] Properties to set
                 */
                function ModifierInfo(properties) {
                    if (properties)
                        for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                            if (properties[keys[i]] != null)
                                this[keys[i]] = properties[keys[i]];
                }

                /**
                 * ModifierInfo domain.
                 * @member {esf.DogmaEffects.DogmaEffect.ModifierInfo.Domain} domain
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.domain = 0;

                /**
                 * ModifierInfo func.
                 * @member {esf.DogmaEffects.DogmaEffect.ModifierInfo.Func} func
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.func = 0;

                /**
                 * ModifierInfo modifiedAttributeID.
                 * @member {number} modifiedAttributeID
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.modifiedAttributeID = 0;

                /**
                 * ModifierInfo modifyingAttributeID.
                 * @member {number} modifyingAttributeID
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.modifyingAttributeID = 0;

                /**
                 * ModifierInfo operation.
                 * @member {number} operation
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.operation = 0;

                /**
                 * ModifierInfo groupID.
                 * @member {number} groupID
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.groupID = 0;

                /**
                 * ModifierInfo skillTypeID.
                 * @member {number} skillTypeID
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 */
                ModifierInfo.prototype.skillTypeID = 0;

                /**
                 * Decodes a ModifierInfo message from the specified reader or buffer.
                 * @function decode
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @static
                 * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
                 * @param {number} [length] Message length if known beforehand
                 * @returns {esf.DogmaEffects.DogmaEffect.ModifierInfo} ModifierInfo
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                ModifierInfo.decode = function decode(reader, length, error) {
                    if (!(reader instanceof $Reader))
                        reader = $Reader.create(reader);
                    let end = length === undefined ? reader.len : reader.pos + length, message = new $root.esf.DogmaEffects.DogmaEffect.ModifierInfo();
                    while (reader.pos < end) {
                        let tag = reader.uint32();
                        if (tag === error)
                            break;
                        switch (tag >>> 3) {
                        case 1: {
                                message.domain = reader.int32();
                                break;
                            }
                        case 2: {
                                message.func = reader.int32();
                                break;
                            }
                        case 3: {
                                message.modifiedAttributeID = reader.int32();
                                break;
                            }
                        case 4: {
                                message.modifyingAttributeID = reader.int32();
                                break;
                            }
                        case 5: {
                                message.operation = reader.int32();
                                break;
                            }
                        case 6: {
                                message.groupID = reader.int32();
                                break;
                            }
                        case 7: {
                                message.skillTypeID = reader.int32();
                                break;
                            }
                        default:
                            reader.skipType(tag & 7);
                            break;
                        }
                    }
                    if (!message.hasOwnProperty("domain"))
                        throw $util.ProtocolError("missing required 'domain'", { instance: message });
                    if (!message.hasOwnProperty("func"))
                        throw $util.ProtocolError("missing required 'func'", { instance: message });
                    return message;
                };

                /**
                 * Creates a ModifierInfo message from a plain object. Also converts values to their respective internal types.
                 * @function fromObject
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @static
                 * @param {Object.<string,*>} object Plain object
                 * @returns {esf.DogmaEffects.DogmaEffect.ModifierInfo} ModifierInfo
                 */
                ModifierInfo.fromObject = function fromObject(object) {
                    if (object instanceof $root.esf.DogmaEffects.DogmaEffect.ModifierInfo)
                        return object;
                    let message = new $root.esf.DogmaEffects.DogmaEffect.ModifierInfo();
                    switch (object.domain) {
                    default:
                        if (typeof object.domain === "number") {
                            message.domain = object.domain;
                            break;
                        }
                        break;
                    case "itemID":
                    case 0:
                        message.domain = 0;
                        break;
                    case "shipID":
                    case 1:
                        message.domain = 1;
                        break;
                    case "charID":
                    case 2:
                        message.domain = 2;
                        break;
                    case "otherID":
                    case 3:
                        message.domain = 3;
                        break;
                    case "structureID":
                    case 4:
                        message.domain = 4;
                        break;
                    case "target":
                    case 5:
                        message.domain = 5;
                        break;
                    case "targetID":
                    case 6:
                        message.domain = 6;
                        break;
                    }
                    switch (object.func) {
                    default:
                        if (typeof object.func === "number") {
                            message.func = object.func;
                            break;
                        }
                        break;
                    case "ItemModifier":
                    case 0:
                        message.func = 0;
                        break;
                    case "LocationGroupModifier":
                    case 1:
                        message.func = 1;
                        break;
                    case "LocationModifier":
                    case 2:
                        message.func = 2;
                        break;
                    case "LocationRequiredSkillModifier":
                    case 3:
                        message.func = 3;
                        break;
                    case "OwnerRequiredSkillModifier":
                    case 4:
                        message.func = 4;
                        break;
                    case "EffectStopper":
                    case 5:
                        message.func = 5;
                        break;
                    }
                    if (object.modifiedAttributeID != null)
                        message.modifiedAttributeID = object.modifiedAttributeID | 0;
                    if (object.modifyingAttributeID != null)
                        message.modifyingAttributeID = object.modifyingAttributeID | 0;
                    if (object.operation != null)
                        message.operation = object.operation | 0;
                    if (object.groupID != null)
                        message.groupID = object.groupID | 0;
                    if (object.skillTypeID != null)
                        message.skillTypeID = object.skillTypeID | 0;
                    return message;
                };

                /**
                 * Creates a plain object from a ModifierInfo message. Also converts values to other types if specified.
                 * @function toObject
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @static
                 * @param {esf.DogmaEffects.DogmaEffect.ModifierInfo} message ModifierInfo
                 * @param {$protobuf.IConversionOptions} [options] Conversion options
                 * @returns {Object.<string,*>} Plain object
                 */
                ModifierInfo.toObject = function toObject(message, options) {
                    if (!options)
                        options = {};
                    let object = {};
                    if (options.defaults) {
                        object.domain = options.enums === String ? "itemID" : 0;
                        object.func = options.enums === String ? "ItemModifier" : 0;
                        object.modifiedAttributeID = 0;
                        object.modifyingAttributeID = 0;
                        object.operation = 0;
                        object.groupID = 0;
                        object.skillTypeID = 0;
                    }
                    if (message.domain != null && message.hasOwnProperty("domain"))
                        object.domain = options.enums === String ? $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.Domain[message.domain] === undefined ? message.domain : $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.Domain[message.domain] : message.domain;
                    if (message.func != null && message.hasOwnProperty("func"))
                        object.func = options.enums === String ? $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.Func[message.func] === undefined ? message.func : $root.esf.DogmaEffects.DogmaEffect.ModifierInfo.Func[message.func] : message.func;
                    if (message.modifiedAttributeID != null && message.hasOwnProperty("modifiedAttributeID"))
                        object.modifiedAttributeID = message.modifiedAttributeID;
                    if (message.modifyingAttributeID != null && message.hasOwnProperty("modifyingAttributeID"))
                        object.modifyingAttributeID = message.modifyingAttributeID;
                    if (message.operation != null && message.hasOwnProperty("operation"))
                        object.operation = message.operation;
                    if (message.groupID != null && message.hasOwnProperty("groupID"))
                        object.groupID = message.groupID;
                    if (message.skillTypeID != null && message.hasOwnProperty("skillTypeID"))
                        object.skillTypeID = message.skillTypeID;
                    return object;
                };

                /**
                 * Converts this ModifierInfo to JSON.
                 * @function toJSON
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @instance
                 * @returns {Object.<string,*>} JSON object
                 */
                ModifierInfo.prototype.toJSON = function toJSON() {
                    return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
                };

                /**
                 * Gets the default type url for ModifierInfo
                 * @function getTypeUrl
                 * @memberof esf.DogmaEffects.DogmaEffect.ModifierInfo
                 * @static
                 * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns {string} The default type url
                 */
                ModifierInfo.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                    if (typeUrlPrefix === undefined) {
                        typeUrlPrefix = "type.googleapis.com";
                    }
                    return typeUrlPrefix + "/esf.DogmaEffects.DogmaEffect.ModifierInfo";
                };

                /**
                 * Domain enum.
                 * @name esf.DogmaEffects.DogmaEffect.ModifierInfo.Domain
                 * @enum {number}
                 * @property {number} itemID=0 itemID value
                 * @property {number} shipID=1 shipID value
                 * @property {number} charID=2 charID value
                 * @property {number} otherID=3 otherID value
                 * @property {number} structureID=4 structureID value
                 * @property {number} target=5 target value
                 * @property {number} targetID=6 targetID value
                 */
                ModifierInfo.Domain = (function() {
                    const valuesById = {}, values = Object.create(valuesById);
                    values[valuesById[0] = "itemID"] = 0;
                    values[valuesById[1] = "shipID"] = 1;
                    values[valuesById[2] = "charID"] = 2;
                    values[valuesById[3] = "otherID"] = 3;
                    values[valuesById[4] = "structureID"] = 4;
                    values[valuesById[5] = "target"] = 5;
                    values[valuesById[6] = "targetID"] = 6;
                    return values;
                })();

                /**
                 * Func enum.
                 * @name esf.DogmaEffects.DogmaEffect.ModifierInfo.Func
                 * @enum {number}
                 * @property {number} ItemModifier=0 ItemModifier value
                 * @property {number} LocationGroupModifier=1 LocationGroupModifier value
                 * @property {number} LocationModifier=2 LocationModifier value
                 * @property {number} LocationRequiredSkillModifier=3 LocationRequiredSkillModifier value
                 * @property {number} OwnerRequiredSkillModifier=4 OwnerRequiredSkillModifier value
                 * @property {number} EffectStopper=5 EffectStopper value
                 */
                ModifierInfo.Func = (function() {
                    const valuesById = {}, values = Object.create(valuesById);
                    values[valuesById[0] = "ItemModifier"] = 0;
                    values[valuesById[1] = "LocationGroupModifier"] = 1;
                    values[valuesById[2] = "LocationModifier"] = 2;
                    values[valuesById[3] = "LocationRequiredSkillModifier"] = 3;
                    values[valuesById[4] = "OwnerRequiredSkillModifier"] = 4;
                    values[valuesById[5] = "EffectStopper"] = 5;
                    return values;
                })();

                return ModifierInfo;
            })();

            return DogmaEffect;
        })();

        return DogmaEffects;
    })();

    return esf;
})();

export { $root as default };
