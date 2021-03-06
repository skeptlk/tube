import { User } from './user'

export class Comment {
    id: number;
    userId: number;
    user: User;
    videoId: number;
    replyTo: number;
    replyCount: number = 0;
    replies: Comment[] = [];
    text: string;
    createdAt: Date;

    constructor (base: any = undefined) {
        if (base) {
            this.id = base['id'];
            this.userId = base['userId'];
            this.user = new User(base['user']);
            this.videoId = base['videoId'];
            this.replyTo = base['replyTo'];
            this.replyCount = base['replyCount'];
            this.text = base['text'];
            this.createdAt = new Date(base['createdAt']);
    
            if (base['replies']) {
                this.replies = base['replies'].map(r => new Comment(r));
            }
        } 
        else this.createdAt = new Date();
    }
}
