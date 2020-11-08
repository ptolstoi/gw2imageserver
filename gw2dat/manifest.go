package gw2dat

/*


   struct FileHeader
   {
       uint8 magic[2];
       uint16 descriptorType;
       uint16 zero;
       uint16 headerSize;
       uint32 contentType;
   };

   struct ChunkHeader
   {
       uint32 magic;
       uint32 nextChunkOffset;
       uint16 version;
       uint16 headerSize;
       uint32 descriptorOffset;
   };

==================================================
 Chunk: ARMF, versions: 2, strucTab: 0x149F238
==================================================
=> Version: 1
typedef struct
{
    dword baseId;
    dword fileId;
    dword size;
    dword flags;
    wchar_ptr name;
} PackAssetManifestFileV1<optimize=false>;

typedef struct
{
    dword baseId;
    dword fileId;
    dword size;
    dword fileType;
} PackAssetExtraFileV1<optimize=false>;

typedef struct
{
    dword buildId;
    TSTRUCT_ARRAY_PTR_START PackAssetManifestFileV1 manifests TSTRUCT_ARRAY_PTR_END;
    TSTRUCT_ARRAY_PTR_START PackAssetExtraFileV1 extraFiles TSTRUCT_ARRAY_PTR_END;
} PackAssetRootManifestV1<optimize=false>;

=> Version: 0
typedef struct
{
    dword baseId;
    dword fileId;
    dword size;
    dword flags;
    wchar_ptr name;
} PackAssetManifestFileV0<optimize=false>;

typedef struct
{
    dword buildId;
    TSTRUCT_ARRAY_PTR_START PackAssetManifestFileV0 manifests TSTRUCT_ARRAY_PTR_END;
} PackAssetRootManifestV0<optimize=false>;



 */