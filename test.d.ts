declare interface ProjectInfo {
    // 模块名称
    name: string;
    // 产品信息
    product: {
        // 产品名称
        name: string;
        // 产品代号
        codeName: string;
        // 产品类别
        type: string;
        // 产品描述
        desc: string;
        // 产品背景
        background: string;
    };
    // 版本信息
    version: {
        // 版本名称
        name: string;
        // 版本描述
        desc: string;
    };
    // 依赖信息
    depends: {
        // 当前依赖列表
        current: {
            // 名称
            name: string;
            // 版本
            version: string;
            // 描述
            desc: string;
            // 元数据
            metadata: {};
        }[];
        group: {
            [key: string]: {
                name: string;
                desc: string;
                depends: {
                    // 名称
                    name: string;
                    // 版本
                    version: string;
                    // 描述
                    desc: string;
                    // 元数据
                    metadata: {};
                }[]
            }
        };
    }
}