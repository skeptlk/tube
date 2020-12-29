import { User } from './user'

export class Comment {
    id: number;
    userId: number;
    user: User;
    videoId: number;
    replyTo: number;
    replyCount: number;
    replies: Comment[];
    text: string;

    constructor (base: any = undefined) {
        if (base) {
            this.id = base['id'];
            this.userId = base['userId'];
            this.user = new User(base['user']);
            this.videoId = base['videoId'];
            this.replyTo = base['replyTo'];
            this.replyCount = base['replyCount'];
            this.text = base['text'];
    
            if (base['replies']) {
                this.replies = base['replies'].map(r => new Comment(r));
            }
        }
    }
}
