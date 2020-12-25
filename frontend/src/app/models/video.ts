import { User } from "./user";

export class Video {
    id: number;
    userId: number;
    user: User;
    title: string;
    description: string;
    duration: number;
    views: number;
    likes: number;
    dislikes: number;
    url: string;
    thumbnail: string;
    isLiked: boolean;
    isDisliked: boolean;

    constructor(base: any) {
        this.id =           base['id'];
        this.user = new User(base['user']);
        this.userId =       base['userId'];
        this.title =        base['title'];
        this.description =  base['description'];
        this.duration =     base['duration'];
        this.views =        base['views'];
        this.likes =        base['likes'];
        this.dislikes =     base['dislikes'];
        this.url =          base['url'];
        this.thumbnail =    base['thumbnail'];
        this.isLiked =      base['isLiked']
        this.isDisliked =   base['isDisliked']
    }
}
