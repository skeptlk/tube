import { Component, Input, OnInit } from '@angular/core';
import { CommentsService } from 'src/app/services';
import { Comment } from '../../models';

@Component({
    selector: 'single-comment',
    templateUrl: './single-comment.component.html',
    styleUrls: ['./single-comment.component.scss']
})
export class SingleCommentComponent {

    @Input() comment: Comment;
    repliesShown: boolean = false;

    constructor(private commentsService: CommentsService) { }

    toggleReplies() {
        this.repliesShown = !this.repliesShown;
        if (this.repliesShown && this.comment.replies.length < this.comment.replyCount) {
            this.commentsService.get(this.comment.id).subscribe(comm => {                
                this.comment.replies = [...comm.replies];
                this.comment.replyCount = comm.replyCount;
            })
        }
    }

    reply() {
        this.commentsService.replyTo = this.comment;
    }

}
